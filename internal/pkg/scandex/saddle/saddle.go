package saddle

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

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

type Option struct {
	InitPoolFunc       InitPoolHandler
	UpdateReservesFunc UpdateReservesHandler
}

type Saddle struct {
	scanDexCfg  *config.ScanDex
	properties  Properties
	scanService *service.ScanService
	option      Option
}

type InitPoolHandler func(ctx context.Context, scanDexCfg *config.ScanDex, properties Properties, scanService *service.ScanService) error
type UpdateReservesHandler func(ctx context.Context, scanDexCfg *config.ScanDex, properties Properties, scanService *service.ScanService)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return NewWithFunc(scanDexCfg, scanService, Option{
		InitPoolFunc:       InitPoolFunc,
		UpdateReservesFunc: UpdateReservesFunc,
	})
}

func NewWithFunc(scanDexCfg *config.ScanDex, scanService *service.ScanService, option Option) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}
	return &Saddle{
		scanDexCfg:  scanDexCfg,
		properties:  properties,
		scanService: scanService,
		option:      option,
	}, nil
}

func (t *Saddle) InitPool(ctx context.Context) error {
	if t.option.InitPoolFunc != nil {
		return t.option.InitPoolFunc(ctx, t.scanDexCfg, t.properties, t.scanService)
	}
	return InitPoolFunc(ctx, t.scanDexCfg, t.properties, t.scanService)
}

func (t *Saddle) UpdateNewPools(ctx context.Context) {
}

func (t *Saddle) UpdateReserves(ctx context.Context) {
	if t.option.UpdateReservesFunc != nil {
		t.option.UpdateReservesFunc(ctx, t.scanDexCfg, t.properties, t.scanService)
	}
	UpdateReservesFunc(ctx, t.scanDexCfg, t.properties, t.scanService)
}

func (t *Saddle) UpdateTotalSupply(ctx context.Context) {
	uniswap.UpdateTotalSupplyJob(ctx,
		t.scanDexCfg,
		t.scanService,
		uniswap.UpdateTotalSupplyHandler,
		t.properties.TotalSupplyJobIntervalSec,
		t.properties.UpdateReserveBulk)
}

func InitPoolFunc(ctx context.Context, scanDexCfg *config.ScanDex, properties Properties, scanService *service.ScanService) error {

	poolsFile, err := os.Open(path.Join(scanService.Config().DataFolder, properties.PoolPath))
	if err != nil {
		logger.Errorf("failed to open config file: %v", err)
		return err
	}
	defer poolsFile.Close()
	byteValue, _ := io.ReadAll(poolsFile)

	var pools []PoolItem
	err = json.Unmarshal(byteValue, &pools)
	if err != nil {
		logger.Errorf("failed to parse pools: %v", err)
	}
	logger.Infof("got %v pools from file: %s", len(pools), path.Join(scanService.Config().DataFolder, properties.PoolPath))
	for i := range pools {
		var pool = pools[i]
		if scanService.ExistPool(ctx, strings.ToLower(pool.ID)) {
			continue
		}
		var swapStorage SwapStorage
		var calls = make([]*repository.CallParams, 0)
		calls = append(calls, &repository.CallParams{
			ABI:    abis.Saddle,
			Target: pool.ID,
			Method: "swapStorage",
			Params: nil,
			Output: &swapStorage,
		})
		if err := scanService.MultiCall(ctx, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			continue
		}
		var tokens = make([]*entity.PoolToken, 0)
		var reserves = make([]string, 0)
		var staticExtra = saddle.PoolStaticExtra{
			LpToken: strings.ToLower(swapStorage.LpToken.Hex()),
		}
		for j, item := range pool.Tokens {
			tokenModel := entity.PoolToken{
				Address:   strings.ToLower(item.Address),
				Weight:    1,
				Swappable: true,
			}
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, item.Precision)
			if _, err := scanService.FetchOrGetToken(ctx, pool.Tokens[j].Address); err != nil {
				return err
			}
			tokens = append(tokens, &tokenModel)
			reserves = append(reserves, "0")
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return err
		}
		var newPool = entity.Pool{
			Address:     strings.ToLower(pool.ID),
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    scanDexCfg.Id,
			Type:        constant.PoolTypes.Saddle,
			Timestamp:   0,
			Reserves:    reserves,
			StaticExtra: string(staticExtraBytes),
			Tokens:      tokens,
		}
		scanService.SavePool(ctx, newPool)
	}
	return nil
}

func UpdateReservesFunc(ctx context.Context, scanDexCfg *config.ScanDex, properties Properties, scanService *service.ScanService) {
	f := func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
		var calls = make([]*repository.TryCallParams, 0)
		var swapStorage = make([]SwapStorage, len(pools))
		var extras = make([]saddle.Extra, len(pools))
		var extraStrings = make([]string, len(pools))
		balances := make([]Balances, len(pools))
		totalSupplies := make([]*big.Int, len(pools))
		pos := make([]int, len(pools))
		for i, pool := range pools {
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.Saddle,
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
				DefaultWithdrawFee: "0",
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
			balances[i] = make([]*big.Int, len(pool.Tokens))
			for j := 0; j < len(balances[i]); j++ {
				balances[i][j] = constant.Zero
			}
			lpToken := pool.GetLpToken()
			if len(lpToken) > 0 {
				pos[i] = len(calls)
				for j := range pool.Tokens {
					calls = append(calls, &repository.TryCallParams{
						ABI:    abis.NerveSwap,
						Target: pool.Address,
						Method: "getTokenBalance",
						Params: []interface{}{uint8(j)},
						Output: &balances[i][j],
					})
				}
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

	uniswap.UpdateReserveJob(
		ctx,
		scanDexCfg,
		scanService,
		f,
		properties.ReserveJobInterval,
		properties.UpdateReserveBulk,
		properties.ConcurrentBatches,
	)
}
