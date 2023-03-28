package firebird

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return uniswap.NewWithFunc(scanDexCfg, scanService, uniswap.Option{
		UpdateReserveFunc:          uniswap.UpdateReservesFunc,
		UpdateNewPoolFunc:          updateNewPoolFunc,
		DexType:                    constant.PoolTypes.Firebird,
		FactoryAbi:                 abis.BiswapFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	})
}
func updateNewPoolFunc(ctx context.Context, scanService *service.ScanService,
	option uniswap.Option, scanDexCfg *config.ScanDex,
	properties interface{},
	pairAddresses []common.Address) error {
	var calls = make([]*repository.CallParams, 0)
	var limit = len(pairAddresses)
	calls = make([]*repository.CallParams, 0)
	var token0Addresses = make([]common.Address, limit)
	var token1Addresses = make([]common.Address, limit)
	swapFees := make([]uint32, limit)
	tokenWeights := make([]struct {
		TokenWeight0 uint32
		TokenWeight1 uint32
	}, limit)
	for i := 0; i < limit; i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.FirebirdPair,
			Target: pairAddresses[i].Hex(),
			Method: "token0",
			Params: nil,
			Output: &token0Addresses[i],
		})
		calls = append(calls, &repository.CallParams{
			ABI:    abis.FirebirdPair,
			Target: pairAddresses[i].Hex(),
			Method: "token1",
			Params: nil,
			Output: &token1Addresses[i],
		})
		calls = append(calls, &repository.CallParams{
			ABI:    abis.PolydexPair,
			Target: pairAddresses[i].Hex(),
			Method: "getSwapFee",
			Params: nil,
			Output: &swapFees[i],
		})
		calls = append(calls, &repository.CallParams{
			ABI:    abis.PolydexPair,
			Target: pairAddresses[i].Hex(),
			Method: "getTokenWeights",
			Params: nil,
			Output: &tokenWeights[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return err
	}

	for i, pair := range pairAddresses {
		p := strings.ToLower(pair.Hex())
		token0Address := strings.ToLower(token0Addresses[i].Hex())
		token1Address := strings.ToLower(token1Addresses[i].Hex())
		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    uint(tokenWeights[i].TokenWeight0),
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    uint(tokenWeights[i].TokenWeight1),
			Swappable: true,
		}
		if _, err := scanService.FetchOrGetToken(ctx, token0.Address); err != nil {
			return err
		}
		if _, err := scanService.FetchOrGetToken(ctx, token1.Address); err != nil {
			return err
		}
		swapFee := float64(swapFees[i]) / 10000
		var pool = entity.Pool{
			Address:    p,
			ReserveUsd: 0,
			SwapFee:    swapFee,
			Exchange:   scanDexCfg.Id,
			Type:       option.DexType,
			Timestamp:  0,
			Reserves:   []string{"0", "0"},
			Tokens:     []*entity.PoolToken{&token0, &token1},
		}
		err := scanService.SavePool(ctx, pool)
		if err != nil {
			logger.Errorf("can not save pool address=%v err=%v", pool.Address, err)
			return err
		}
	}
	return nil
}
