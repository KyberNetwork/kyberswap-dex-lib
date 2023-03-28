package uniswapv3

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Option struct {
	DexType string
}

type UniSwapV3 struct {
	scanDexCfg        *config.ScanDex
	scanService       *service.ScanService
	graphqlClient     *graphql.Client
	properties        Properties
	option            Option
	preGenesisPoolIDs []string
}

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return NewWithFunc(scanDexCfg, scanService, Option{
		DexType: constant.PoolTypes.UniV3,
	})
}
func NewWithFunc(scanDexCfg *config.ScanDex, scanService *service.ScanService, option Option) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	// Initialize graphql client with custom HTTP client (use custom timeout instead of 0)
	// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	httpClient := &http.Client{
		Timeout: defaultGraphQLRequestTimeout,
	}
	graphqlClient := graphql.NewClient(properties.SubgraphAPI, graphql.WithHTTPClient(httpClient))

	return &UniSwapV3{
		scanDexCfg:        scanDexCfg,
		scanService:       scanService,
		graphqlClient:     graphqlClient,
		properties:        properties,
		option:            option,
		preGenesisPoolIDs: nil,
	}, nil
}

func (t *UniSwapV3) InitPool(ctx context.Context) error {
	if t.properties.PreGenesisPoolPath == "" {
		return nil
	}

	poolsFile, err := os.Open(path.Join(t.scanService.Config().DataFolder, t.properties.PreGenesisPoolPath))
	if err != nil {
		logger.Errorf("failed to open config file: %v", err)
		return err
	}
	defer poolsFile.Close()
	byteValue, _ := io.ReadAll(poolsFile)

	var pools []preGenesisPool
	err = json.Unmarshal(byteValue, &pools)
	if err != nil {
		logger.Errorf("failed to parse pools: %v", err)
	}
	logger.Infof("got %v pools from file: %s", len(pools), path.Join(t.scanService.Config().DataFolder, t.properties.PreGenesisPoolPath))

	for _, p := range pools {
		t.preGenesisPoolIDs = append(t.preGenesisPoolIDs, p.ID)
	}

	return nil
}

func (t *UniSwapV3) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, first, skip int) ([]SubgraphPool, error) {
	allowSubgraphError := utils.IsAllowSubgraphError()

	req := graphql.NewRequest(getPoolsListQuery(allowSubgraphError, lastCreatedAtTimestamp, first, skip))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := t.graphqlClient.Run(ctx, req, &response); err != nil {
		// Workaround at the moment to live with the error subgraph
		if allowSubgraphError && response.Pools != nil && len(response.Pools) > 0 {
			return response.Pools, nil
		}

		logger.Errorf("failed to query subgraph, err: %v", err)
		return nil, err
	}

	return response.Pools, nil
}

func (t *UniSwapV3) UpdateNewPools(ctx context.Context) {
	const limit = 1000
	offsetKey := utils.Join(t.scanDexCfg.Id, "offset")

	run := func() error {
		offset, err := t.scanService.GetLastDexOffset(ctx, offsetKey)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}

		lastCreatedAtTimestamp := big.NewInt(int64(offset))

		subgraphPools, err := t.getPoolsList(ctx, lastCreatedAtTimestamp, limit, 0)

		if err != nil {
			return err
		}

		numPools := len(subgraphPools)

		logger.Infof("got %v subgraphPools from subgraph of Uniswap V3", numPools)

		for _, p := range subgraphPools {
			if t.scanService.ExistPool(ctx, p.ID) {
				continue
			}

			var tokens = make([]*entity.PoolToken, 0)
			var reserves = make([]string, 0)
			var staticField = StaticExtra{
				PoolId: p.ID,
			}

			if p.Token0.Address != "" {
				token0Decimals, err := strconv.Atoi(p.Token0.Decimals)

				if err != nil {
					token0Decimals = 18
				}

				tokenModel := entity.PoolToken{
					Address:   p.Token0.Address,
					Name:      p.Token0.Name,
					Symbol:    p.Token0.Symbol,
					Decimals:  uint8(token0Decimals),
					Weight:    50,
					Swappable: true,
				}

				if _, err := t.scanService.FetchOrGetToken(ctx, tokenModel.Address); err != nil {
					logger.Errorf("failed to fetch or get token %v, err: %+v", tokenModel.Address, err)
					return err
				}

				tokens = append(tokens, &tokenModel)
				reserves = append(reserves, "0")
			}

			if p.Token1.Address != "" {
				token1Decimals, err := strconv.Atoi(p.Token1.Decimals)

				if err != nil {
					token1Decimals = 18
				}

				tokenModel := entity.PoolToken{
					Address:   p.Token1.Address,
					Name:      p.Token1.Name,
					Symbol:    p.Token1.Symbol,
					Decimals:  uint8(token1Decimals),
					Weight:    50,
					Swappable: true,
				}

				if _, err := t.scanService.FetchOrGetToken(ctx, tokenModel.Address); err != nil {
					logger.Errorf("failed to fetch or get token %v, err: %+v", tokenModel.Address, err)
					return err
				}

				tokens = append(tokens, &tokenModel)
				reserves = append(reserves, "0")
			}

			var swapFee, _ = strconv.ParseFloat(p.FeeTier, 64)

			staticBytes, _ := json.Marshal(staticField)
			var newPool = entity.Pool{
				Address:      p.ID,
				ReserveUsd:   0,
				AmplifiedTvl: 0,
				SwapFee:      swapFee,
				Exchange:     t.scanDexCfg.Id,
				Type:         constant.PoolTypes.UniV3,
				Timestamp:    0,
				Reserves:     reserves,
				Tokens:       tokens,
				StaticExtra:  string(staticBytes),
			}

			err = t.scanService.SavePool(ctx, newPool)

			if err != nil {
				logger.Errorf("can not save pool address=%v err=%+v", p.ID, err)
				return err
			}

			err = t.scanService.SetLastDexOffset(ctx, offsetKey, p.CreatedAtTimestamp)
			if err != nil {
				logger.Errorf("can not save config pair offset to database err %v", err)
				return err
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

/**
 * 	Build poolIds array in string format
 */
//func getPoolIdsString(pools []entity.Pool) string {
//	poolIds := "["
//
//	for i, p := range pools {
//		if i == len(pools)-1 {
//			poolIds += fmt.Sprintf("\"%s\"", p.Address)
//			break
//		}
//
//		poolIds += fmt.Sprintf("\"%s\",", p.Address)
//	}
//
//	poolIds += "]"
//
//	return poolIds
//}
//
//func (t *UniSwapV3) getMultiplePoolsTicks(ctx context.Context, pools []entity.Pool) ([]SubgraphPoolTicks, error) {
//	poolIds := getPoolIdsString(pools)
//
//	req := graphql.NewRequest(fmt.Sprintf(`{
//		pools(where : {id_in: %v}) {
//			id
//			ticks(first: 1000) {
//				tickIdx
//				liquidityNet
//				liquidityGross
//			}
//		}
//	}`, poolIds),
//	)
//
//	var response struct {
//		Pools []SubgraphPoolTicks `json:"pools"`
//	}
//
//	if err := t.graphqlClient.Run(ctx, req, &response); err != nil {
//		logger.Errorf("failed to query subgraph, err: %v", err)
//		return nil, err
//	}
//
//	return response.Pools, nil
//}

/**
 * Get all ticks of a pool
 * Some pools have more than 1000 ticks, subgraph API has a limit of 1000 results per query, so getMultiplePoolsTicks function does not work
 */
func (t *UniSwapV3) getPoolTicks(ctx context.Context, pool entity.Pool) ([]TickResp, error) {
	defer func() {
		logger.WithFields(logger.Fields{
			"pool": pool.Address,
		}).Debug("done fetching pool ticks from subgraph")
	}()

	allowSubgraphError := utils.IsAllowSubgraphError()

	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(getPoolTicksQuery(allowSubgraphError, pool.Address, skip))

		var resp struct {
			Pool *SubgraphPoolTicks `json:"pool"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph
			if allowSubgraphError && resp.Pool == nil {
				logger.Errorf("failed to query subgraph, err: %v", err)
				return nil, err
			}
		}

		if resp.Pool == nil || resp.Pool.Ticks == nil || len(resp.Pool.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Pool.Ticks...)

		if len(resp.Pool.Ticks) < graphFirstLimit {
			break
		}

		skip += len(resp.Pool.Ticks)
		if skip > graphSkipLimit {
			logger.Infoln("hit skip limit, continue in next cycle")
			break
		}
	}

	return ticks, nil
}

func (t *UniSwapV3) UpdateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("func UpdateReservesFunc recovered, error: %v", r)
		}

		logger.Debug("finished UpdateReservesFunc")
	}()

	var calls = make([]*repository.TryCallParams, 0)

	liquidity := make([]*big.Int, len(pools))
	slot0 := make([]Slot0, len(pools))
	reserve0Array := make([]*big.Int, len(pools))
	reserve1Array := make([]*big.Int, len(pools))

	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.UniswapV3Pool,
			Target: pool.Address,
			Method: "liquidity",
			Params: nil,
			Output: &liquidity[i],
		})

		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.UniswapV3Pool,
			Target: pool.Address,
			Method: "slot0",
			Params: nil,
			Output: &slot0[i],
		})

		// Get reserves of 2 tokens in pool to calculate TvL
		if len(pool.Tokens) == 2 {
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.ERC20,
				Target: pool.Tokens[0].Address,
				Method: "balanceOf",
				Params: []interface{}{common.HexToAddress(pool.Address)},
				Output: &reserve0Array[i],
			})

			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.ERC20,
				Target: pool.Tokens[1].Address,
				Method: "balanceOf",
				Params: []interface{}{common.HexToAddress(pool.Address)},
				Output: &reserve1Array[i],
			})
		}
	}

	if err := scanService.TryAggregateForce(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}

	logger.Debug("done fetching data through multicall")

	poolsTicksMap := make(map[string][]Tick)

	for _, p := range pools {
		var poolTicks []TickResp
		var err error

		if utils.StringContains(t.preGenesisPoolIDs, p.Address) {
			poolTicks, err = t.getPoolTicksFromSC(ctx, p)
			if err != nil {
				logger.Errorf("failed to call SC for pool ticks, pool: %v, err: %v", p.Address, err)
				continue
			}
		} else {
			poolTicks, err = t.getPoolTicks(ctx, p)
			if err != nil {
				logger.Errorf("failed to query subgraph for pool ticks, pool: %v, err: %v", p.Address, err)
				continue
			}
		}

		var ticks []Tick
		for _, tickResp := range poolTicks {
			tick, err := transformTickRespToTick(tickResp)
			if err != nil {
				logger.Errorf("failed to transform tickResp to tick for pool: %v, err: %v", p.Address, err)
				continue
			}

			ticks = append(ticks, tick)
		}

		poolsTicksMap[p.Address] = ticks
	}

	var ret = 0
	for i, p := range pools {
		extraBytes, _ := json.Marshal(Extra{
			Liquidity:    liquidity[i],
			SqrtPriceX96: slot0[i].SqrtPriceX96,
			Tick:         slot0[i].Tick,
			Ticks:        poolsTicksMap[p.Address],
		})

		extra := string(extraBytes)
		err := scanService.UpdatePoolExtra(ctx, p.Address, extra)

		if err != nil {
			logger.Errorf("failed to update extra for pool: %v, err %v", p.Address, err)
			continue
		}

		logger.WithFields(logger.Fields{
			"pool": p.Address,
		}).Debug("done updating pool extra")

		if reserve0Array[i] == nil {
			reserve0Array[i] = big.NewInt(0)
		}

		if reserve1Array[i] == nil {
			reserve1Array[i] = big.NewInt(0)
		}

		err = scanService.UpdatePoolReserve(ctx, p.Address, time.Now().Unix(), []string{
			reserve0Array[i].String(),
			reserve1Array[i].String(),
		})

		if err != nil {
			logger.Errorf("failed to update reserve for pool: %v, err %v", p.Address, err)
		} else {
			ret++
		}

		logger.WithFields(logger.Fields{
			"pool": p.Address,
		}).Debug("done updating pool reserve")
	}

	return ret
}

func (t *UniSwapV3) UpdateReserves(ctx context.Context) {
	run := func() error {
		sum := int32(0)
		startTime := time.Now()

		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("func UpdateReserves recovered, error: %v", r)
			}

			executionTime := time.Since(startTime)

			logger.
				WithFields(logger.Fields{
					"dex":               t.scanDexCfg.Id,
					"poolsUpdatedCount": sum,
					"duration":          executionTime.Milliseconds(),
				}).
				Info("finished UpdateReserves")

			metrics.HistogramScannerUpdateReservesDuration(executionTime, t.scanDexCfg.Id, int(sum))
		}()

		logger.Infof("updating reserves...")

		updateReserveBulk := t.properties.UpdateReserveBulk
		pools := t.scanService.GetPoolIdsByExchange(ctx, t.scanDexCfg.Id)
		var wg sync.WaitGroup
		concurrentGoroutines := make(chan struct{}, t.properties.ConcurrentBatches)

		for i := 0; i < len(pools); i += updateReserveBulk {
			wg.Add(1)
			endIndex := i + updateReserveBulk
			if endIndex > len(pools) {
				endIndex = len(pools)
			}

			go func(startIndex, endIndex int) {
				defer func() {
					wg.Done()
					<-concurrentGoroutines

					logger.Debug("release goroutine!")
				}()
				concurrentGoroutines <- struct{}{}

				pools, err := t.scanService.GetPoolsByAddresses(ctx, pools[startIndex:endIndex])
				if err != nil {
					logger.Errorf("failed to get pools by addresses, err: ", err)
					return
				}

				count := t.UpdateReservesFunc(ctx, t.scanService, pools)

				atomic.AddInt32(&sum, int32(count))

				logger.WithFields(logger.Fields{
					"poolsUpdatedCount": count,
				}).Debug("finished goroutine inside UpdateReserves")
			}(i, endIndex)
		}

		logger.Debug("WaitGroup is waiting ...")

		wg.Wait()

		logger.Infof("update reserves %v pools in %v", sum, time.Since(startTime))

		return nil
	}

	for {
		if err := run(); err != nil {
			logger.Errorf("failed to update reserves err: %v", err)
		}

		time.Sleep(t.properties.ReserveJobInterval.Duration)
	}
}

func (t *UniSwapV3) UpdateTotalSupply(ctx context.Context) {

}
