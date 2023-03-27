package uniswap

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/metrics"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/duration"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type Option struct {
	UpdateReserveFunc          UpdateReserveHandler
	UpdateNewPoolFunc          UpdateNewPoolHandler
	DexType                    string
	FactoryAbi                 abi.ABI
	FactoryPairCountMethodCall string
	FactoryGetPairMethodCall   string
}
type UniSwap struct {
	scanDexCfg  *config.ScanDex
	scanService *service.ScanService
	properties  Properties
	option      Option
}

type UpdateReserveHandler func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int
type UpdateNewPoolHandler func(ctx context.Context, scanService *service.ScanService, option Option, scanDexCfg *config.ScanDex, properties interface{}, pairAddresses []common.Address) error

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return NewWithFunc(scanDexCfg, scanService, Option{
		UpdateReserveFunc:          UpdateReservesFunc,
		UpdateNewPoolFunc:          UpdateNewPoolFunc,
		DexType:                    constant.PoolTypes.Uni,
		FactoryAbi:                 abis.BiswapFactory,
		FactoryGetPairMethodCall:   "allPairs",
		FactoryPairCountMethodCall: "allPairsLength",
	})
}
func NewWithFunc(scanDexCfg *config.ScanDex, scanService *service.ScanService, option Option) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}
	return &UniSwap{
		scanDexCfg,
		scanService,
		properties,
		option,
	}, nil
}

func (t *UniSwap) InitPool(ctx context.Context) error {
	return nil
}

func UpdateNewPoolFunc(ctx context.Context, scanService *service.ScanService,
	option Option, scanDexCfg *config.ScanDex,
	properties interface{},
	pairAddresses []common.Address) error {
	var calls = make([]*repository.CallParams, 0)
	var limit = len(pairAddresses)
	calls = make([]*repository.CallParams, 0)
	var token0Addresses = make([]common.Address, limit)
	var token1Addresses = make([]common.Address, limit)
	for i := 0; i < limit; i++ {
		calls = append(calls, &repository.CallParams{
			ABI:    abis.SushiswapPair,
			Target: pairAddresses[i].Hex(),
			Method: "token0",
			Params: nil,
			Output: &token0Addresses[i],
		})
		calls = append(calls, &repository.CallParams{
			ABI:    abis.SushiswapPair,
			Target: pairAddresses[i].Hex(),
			Method: "token1",
			Params: nil,
			Output: &token1Addresses[i],
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
		if scanService.ExistPool(ctx, p) {
			continue
		}
		var token0 = entity.PoolToken{
			Address:   token0Address,
			Weight:    50,
			Swappable: true,
		}
		var token1 = entity.PoolToken{
			Address:   token1Address,
			Weight:    50,
			Swappable: true,
		}
		if _, err := scanService.FetchOrGetToken(ctx, token0.Address); err != nil {
			return err
		}
		if _, err := scanService.FetchOrGetToken(ctx, token1.Address); err != nil {
			return err
		}

		var pool = entity.Pool{
			Address:    p,
			ReserveUsd: 0,
			SwapFee:    properties.(Properties).SwapFee,
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
func (t *UniSwap) UpdateNewPools(ctx context.Context) {

	offsetKey := utils.Join(t.scanDexCfg.Id, "offset")
	run := func() error {
		bulk := t.properties.NewPoolBulk
		offset, err := t.scanService.GetLastDexOffset(ctx, offsetKey)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}

		var lengthBI *big.Int
		err = t.scanService.Call(ctx, &repository.CallParams{
			ABI:    t.option.FactoryAbi,
			Target: t.properties.FactoryAddress,
			Method: t.option.FactoryPairCountMethodCall,
			Params: nil,
			Output: &lengthBI,
		})
		if err != nil {
			return err
		}
		var length = int(lengthBI.Int64())
		for i := offset; i < length; i += bulk {
			l := bulk
			if i+bulk > length {
				l = length - i
			}
			var calls = make([]*repository.CallParams, 0)

			var pairAddresses = make([]common.Address, l)
			for j := 0; j < l; j++ {
				calls = append(calls, &repository.CallParams{
					ABI:    t.option.FactoryAbi,
					Target: t.properties.FactoryAddress,
					Method: t.option.FactoryGetPairMethodCall,
					Params: []interface{}{big.NewInt(int64(i + j))},
					Output: &pairAddresses[j],
				})
			}
			if err := t.scanService.MultiCall(ctx, calls); err != nil {
				logger.Errorf("failed to process multicall, err: %v", err)
				return err
			}
			if err = t.option.UpdateNewPoolFunc(
				ctx, t.scanService, t.option, t.scanDexCfg, t.properties, pairAddresses); err != nil {
				logger.Errorf("failed to process update new pool, err: %v", err)
				return err
			}

			err = t.scanService.SetLastDexOffset(ctx, offsetKey, i+l)
			if err != nil {
				logger.Errorf("can not save config pair offset to database err %v", err)
				return err
			}
			if l > 0 {
				logger.Infof("scan pair size %v %d/%d", l, i+l, length)
			}
		}
		return err
	}
	for {
		err := run()
		if err != nil {
			logger.Errorf("can not update new pool %v", err)
		}
		time.Sleep(time.Duration(t.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (t *UniSwap) UpdateReserves(ctx context.Context) {
	UpdateReserveJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		t.option.UpdateReserveFunc,
		t.properties.ReserveJobInterval,
		t.properties.UpdateReserveBulk,
		t.properties.ConcurrentBatches,
	)
}

func UpdateTotalSupplyHandler(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {

	var calls = make([]*repository.CallParams, 0)
	var totalSupplies = make([]*big.Int, len(pools))
	for i, pool := range pools {
		lpToken := pool.GetLpToken()
		_, err := scanService.FetchOrGetTokenType(ctx, lpToken, pool.Exchange, pool.Address)
		if err != nil {
			return 0
		}
		calls = append(calls, &repository.CallParams{
			ABI:    abis.ERC20,
			Target: lpToken,
			Method: "totalSupply",
			Params: nil,
			Output: &totalSupplies[i],
		})
	}
	if err := scanService.MultiCall(ctx, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	var ret = 0
	for i, pool := range pools {
		err := scanService.UpdatePoolSupply(ctx, pool.Address, totalSupplies[i].String())
		if err != nil {
			logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
		} else {
			ret++
		}
	}
	return ret
}
func (t *UniSwap) UpdateTotalSupply(ctx context.Context) {
	UpdateTotalSupplyJob(ctx,
		t.scanDexCfg,
		t.scanService,
		UpdateTotalSupplyHandler,
		t.properties.TotalSupplyJobIntervalSec,
		t.properties.UpdateReserveBulk)
}

func UpdateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {

	var calls = make([]*repository.TryCallParams, 0)
	reserves := make([]Reserves, len(pools))

	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.SushiswapPair,
			Target: pool.Address,
			Method: "getReserves",
			Params: nil,
			Output: &reserves[i],
		})
	}
	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}
	var ret = 0
	for i, pool := range pools {
		reserve := reserves[i]
		if *calls[i].Success {
			err := scanService.UpdatePoolReserve(ctx, pool.Address, int64(reserve.BlockTimestampLast), []string{
				reserve.Reserve0.String(),
				reserve.Reserve1.String(),
			})
			if err != nil {
				logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			} else {
				ret++
			}

		}
		if !*calls[i].Success {
			logger.Errorf("failed to get reserve: %v", pool.Address)
		}
	}
	return ret
}

func UpdateReserveJob(
	ctx context.Context,
	scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
	updateReserveFunc UpdateReserveHandler,
	reserveJobInterval duration.Duration,
	updateReserveBulk int,
	concurrentBatches int,
) {
	run := func() error {
		sum := int32(0)
		startTime := time.Now()
		defer func() {
			executionTime := time.Since(startTime)

			logger.
				WithFields(logger.Fields{
					"dex":               scanDexCfg.Id,
					"poolsUpdatedCount": sum,
					"duration":          executionTime.Milliseconds(),
				}).
				Info("finished UpdateReserves")

			metrics.HistogramScannerUpdateReservesDuration(executionTime, scanDexCfg.Id, int(sum))
		}()

		pools := scanService.GetPoolIdsByExchange(ctx, scanDexCfg.Id)
		var wg sync.WaitGroup
		concurrentGoroutines := make(chan struct{}, concurrentBatches)

		for i := 0; i < len(pools); i += updateReserveBulk {
			wg.Add(1)
			end := i + updateReserveBulk
			if end > len(pools) {
				end = len(pools)
			}
			go func(s, e int) {
				defer func() {
					wg.Done()
					<-concurrentGoroutines
				}()
				concurrentGoroutines <- struct{}{}
				pools, err := scanService.GetPoolsByAddresses(ctx, pools[s:e])
				if err != nil {
					logger.Errorf(err.Error())
					return
				}
				count := updateReserveFunc(ctx, scanService, pools)
				atomic.AddInt32(&sum, int32(count))

			}(i, end)
		}
		wg.Wait()
		logger.Infof("update reserves %v pairs in %v", sum, time.Since(startTime))
		return nil
	}
	for {
		err := run()
		if err != nil {
			logger.Errorf("can not update reserve err=%v", err)
		}
		time.Sleep(reserveJobInterval.Duration)
	}
}

func UpdateTotalSupplyJob(ctx context.Context, scanDexCfg *config.ScanDex, scanService *service.ScanService, handler UpdateReserveHandler, intervalSec int64, bulk int) {
	run := func() error {
		startTime := time.Now()
		pools := scanService.GetPoolIdsByExchange(ctx, scanDexCfg.Id)
		sum := int32(0)

		for i := 0; i < len(pools); i += bulk {
			end := i + bulk
			if end > len(pools) {
				end = len(pools)
			}

			pools, err := scanService.GetPoolsByAddresses(ctx, pools[i:end])
			if err != nil {
				logger.Errorf(err.Error())
				return err
			}
			count := handler(ctx, scanService, pools)
			atomic.AddInt32(&sum, int32(count))

		}
		logger.Infof("update total supply %v pairs in %v", sum, time.Since(startTime))
		return nil
	}
	for {
		err := run()
		if err != nil {
			logger.Errorf("can not update total supply err=%v", err)
		}
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
}
