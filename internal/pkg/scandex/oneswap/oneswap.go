package oneswap

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/saddle"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return uniswap.NewWithFunc(scanDexCfg, scanService, uniswap.Option{
		UpdateReserveFunc:          updateReservesFunc,
		UpdateNewPoolFunc:          updateNewPoolFunc,
		DexType:                    constant.PoolTypes.Saddle,
		FactoryAbi:                 abis.FirebirdOneSwapFactory,
		FactoryGetPairMethodCall:   "allPools",
		FactoryPairCountMethodCall: "allPoolsLength",
	})
}

func updateNewPoolFunc(ctx context.Context, scanService *service.ScanService,
	option uniswap.Option, scanDexCfg *config.ScanDex,
	properties interface{},
	pairAddresses []common.Address) error {
	var multipliers = make([][]*big.Int, len(pairAddresses))
	var tokenAddresses = make([][]common.Address, len(pairAddresses))
	var swapStorage = make([]SwapStorage, len(pairAddresses))
	var calls = make([]*repository.TryCallParams, 0)
	for i, pairAddress := range pairAddresses {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.FirebirdOneSwap,
			Target: pairAddress.Hex(),
			Method: "getTokenPrecisionMultipliers",
			Params: nil,
			Output: &multipliers[i],
		})
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.FirebirdOneSwap,
			Target: pairAddress.Hex(),
			Method: "getPoolTokens",
			Params: nil,
			Output: &tokenAddresses[i],
		})
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.FirebirdOneSwap,
			Target: pairAddress.Hex(),
			Method: "swapStorage",
			Params: nil,
			Output: &swapStorage[i],
		})
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}
	for i, pairAddress := range pairAddresses {
		var tokens = make([]*entity.PoolToken, 0)
		var reserves = make([]string, 0)
		staticExtra := saddle.PoolStaticExtra{
			LpToken: strings.ToLower(swapStorage[i].LpToken.Hex()),
		}
		for j, item := range tokenAddresses[i] {
			tokenAddress := strings.ToLower(item.Hex())
			if _, err := scanService.FetchOrGetToken(ctx, tokenAddress); err != nil {
				return err
			}
			tokenModel := entity.PoolToken{
				Address:   tokenAddress,
				Weight:    1,
				Swappable: true,
			}
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, multipliers[i][j].String())
			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, "0")
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return err
		}
		var newPool = entity.Pool{
			Address:     strings.ToLower(pairAddress.Hex()),
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    scanDexCfg.Id,
			Type:        option.DexType,
			StaticExtra: string(staticExtraBytes),
			Timestamp:   0,
			Reserves:    reserves,
			Tokens:      tokens,
		}
		scanService.SavePool(ctx, newPool)
	}
	return nil
}
func updateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var calls = make([]*repository.TryCallParams, 0)
	var swapStorage = make([]SwapStorage, len(pools))
	var extras = make([]saddle.Extra, len(pools))
	var extraStrings = make([]string, len(pools))
	balances := make([]Balances, len(pools))
	totalSupplies := make([]*big.Int, len(pools))
	pos := make([]int, len(pools))
	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.FirebirdOneSwap,
			Target: pool.Address,
			Method: "swapStorage",
			Params: nil,
			Output: &swapStorage[i],
		})
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	for i := range pools {
		extra := saddle.Extra{
			InitialA:           swapStorage[i].InitialA.String(),
			FutureA:            swapStorage[i].FutureA.String(),
			InitialATime:       swapStorage[i].InitialATime.Int64(),
			FutureATime:        swapStorage[i].FutureATime.Int64(),
			SwapFee:            swapStorage[i].SwapFee.String(),
			AdminFee:           swapStorage[i].AdminFee.String(),
			DefaultWithdrawFee: swapStorage[i].DefaultWithdrawFee.String(),
			//LpToken:            strings.ToLower(swapStorage[i].LpToken.Hex()),
		}
		extras[i] = extra
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.Errorf("failed to encode extra data, err: %v", err)
			return 0
		}
		extraStrings[i] = string(extraBytes)

	}
	for i, pool := range pools {
		pos[i] = -1
		balances[i] = make([]*big.Int, len(pool.Tokens)+1)
		for j := 0; j < len(balances[i]); j++ {
			balances[i][j] = constant.Zero
		}
		lpToken := pool.GetLpToken()
		if len(lpToken) > 0 {
			pos[i] = len(calls)
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.FirebirdOneSwap,
				Target: pool.Address,
				Method: "getBalances",
				Params: nil,
				Output: &balances[i],
			})
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.ERC20,
				Target: lpToken,
				Method: "totalSupply",
				Params: nil,
				Output: &totalSupplies[i],
			})
		}
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	var ret = 0
	for i, pool := range pools {
		if pos[i] >= 0 && *calls[pos[i]].Success && *calls[pos[i]+1].Success {
			balance := balances[i]
			reserves := make([]string, len(balance))
			for j := 0; j < len(balance); j++ {
				reserves[j] = balance[j].String()
			}
			reserves = append(reserves, totalSupplies[i].String())
			scanService.UpdatePoolExtra(ctx, pool.Address, extraStrings[i])
			scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), reserves)
			ret++
		}
	}
	return ret
}
