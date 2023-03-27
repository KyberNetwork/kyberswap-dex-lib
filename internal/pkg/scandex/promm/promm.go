package promm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/core"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type Option struct {
	DexType string
}

type ProMM struct {
	scanDexCfg    *config.ScanDex
	scanService   *service.ScanService
	graphqlClient *graphql.Client
	properties    Properties
	option        Option
}

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	return NewWithFunc(scanDexCfg, scanService, Option{
		DexType: constant.PoolTypes.ProMM,
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

	return &ProMM{
		scanDexCfg:    scanDexCfg,
		scanService:   scanService,
		graphqlClient: graphqlClient,
		properties:    properties,
		option:        option,
	}, nil
}

func (t *ProMM) InitPool(ctx context.Context) error {
	return nil
}

func (t *ProMM) getPoolsList(ctx context.Context, lastCreatedAtTimestamp *big.Int, first, skip int) ([]SubgraphPool, error) {
	req := graphql.NewRequest(fmt.Sprintf(`{
		pools(where : {createdAtTimestamp_gte: %v}, first: %v, skip: %v, orderBy: createdAtTimestamp, orderDirection: asc) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			feeTier
			token0 {
				id
				name
				symbol
			  	decimals
			}
			token1 {
				id
				name
				symbol
			  	decimals
			}
		}
	}`, lastCreatedAtTimestamp, first, skip),
	)

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}

	if err := t.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.Errorf("failed to query subgraph, err: %v", err)
		return nil, err
	}

	return response.Pools, nil
}

func (t *ProMM) UpdateNewPools(ctx context.Context) {

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

		logger.Infof("got %v subgraphPools from subgraph of ProMM", numPools)

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
				Type:         constant.PoolTypes.ProMM,
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

//func (t *ProMM) getMultiplePoolsTicks(ctx context.Context, pools []entity.Pool) ([]SubgraphPoolTicks, error) {
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
func (t *ProMM) getPoolTicks(ctx context.Context, pool entity.Pool) ([]TickResp, error) {
	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(fmt.Sprintf(`{
		pool(id: "%v") {
			id
			ticks(first: 1000, skip: %v) {
				tickIdx
				liquidityNet
				liquidityGross
			}
		}
	}`, pool.Address, skip),
		)

		var resp struct {
			Pool *SubgraphPoolTicks `json:"pool"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			logger.Errorf("failed to query subgraph, err: %v", err)
			return nil, err
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

func (t *ProMM) UpdateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {

	var calls = make([]*repository.TryCallParams, 0)

	LiquidityStates := make([]LiquidityState, len(pools))
	poolStates := make([]PoolState, len(pools))
	reserve0Array := make([]*big.Int, len(pools))
	reserve1Array := make([]*big.Int, len(pools))

	for i, pool := range pools {
		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.ProMMPool,
			Target: pool.Address,
			Method: "getLiquidityState",
			Params: nil,
			Output: &LiquidityStates[i],
		})

		calls = append(calls, &repository.TryCallParams{
			ABI:    abis.ProMMPool,
			Target: pool.Address,
			Method: "getPoolState",
			Params: nil,
			Output: &poolStates[i],
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

	poolsTicksMap := make(map[string][]Tick)

	for _, pool := range pools {
		poolTicks, err := t.getPoolTicks(ctx, pool)

		if err != nil {
			logger.Errorf("failed to query subgraph for pool ticks, err: %v", err)
			continue
		}

		var ticks []Tick

		for _, t := range poolTicks {
			liquidityGross := new(big.Int)
			liquidityGross, ok := liquidityGross.SetString(t.LiquidityGross, 10)
			if !ok {
				logger.Errorf("Can not convert liquidityGross string to big int for pool: %v, tick: %v", pool.Address, t.TickIdx)
				continue
			}

			liquidityNet := new(big.Int)
			liquidityNet, ok = liquidityNet.SetString(t.LiquidityNet, 10)
			if !ok {
				logger.Errorf("Can not convert liquidityNet string to big int for pool: %v, tick: %v", pool.Address, t.TickIdx)
				continue
			}

			tickIdx, err := strconv.Atoi(t.TickIdx)

			if err != nil {
				logger.Errorf("Can not convert tickIdx string to int for pool: %v, tick: %v", pool.Address, t.TickIdx)
				continue
			}

			ticks = append(ticks, Tick{Index: tickIdx, LiquidityNet: liquidityNet, LiquidityGross: liquidityGross})
		}

		poolsTicksMap[pool.Address] = ticks
	}

	var ret = 0
	for i, pool := range pools {
		extraBytes, _ := json.Marshal(Extra{
			Liquidity:     LiquidityStates[i].BaseL,
			ReinvestL:     LiquidityStates[i].ReinvestL,
			ReinvestLLast: LiquidityStates[i].ReinvestLLast,
			SqrtPriceX96:  poolStates[i].SqrtP,
			Tick:          poolStates[i].CurrentTick,
			Ticks:         poolsTicksMap[pool.Address],
		})

		extra := string(extraBytes)
		err := scanService.UpdatePoolExtra(ctx, pool.Address, extra)

		if err != nil {
			logger.Errorf("failed to update extra for pool: %v, err %v", pool.Address, err)
			continue
		}

		if reserve0Array[i] == nil {
			reserve0Array[i] = big.NewInt(0)
		}

		if reserve1Array[i] == nil {
			reserve1Array[i] = big.NewInt(0)
		}

		err = scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), []string{
			reserve0Array[i].String(),
			reserve1Array[i].String(),
		})

		if err != nil {
			logger.Errorf("failed to update reserve for pool: %v, err %v", pool.Address, err)
		} else {
			ret++
		}
	}

	return ret
}

func (t *ProMM) UpdateReserves(ctx context.Context) {
	uniswap.UpdateReserveJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		t.UpdateReservesFunc,
		t.properties.ReserveJobInterval,
		t.properties.UpdateReserveBulk,
		t.properties.ConcurrentBatches,
	)
}

func (t *ProMM) UpdateTotalSupply(ctx context.Context) {

}
