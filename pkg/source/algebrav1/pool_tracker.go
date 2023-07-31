package algebrav1

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphql.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	graphqlClient := graphqlPkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.Infof("[%v] Start getting new state of pool: %v", d.config.DexID, p.Address)

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to fetch data from RPC")

		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		poolTicks, err = d.getPoolTicks(ctx, p.Address)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to query subgraph for pool ticks")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to fetch pool state, pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	ticks := make([]v3Entities.Tick, 0, len(poolTicks))
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to transform tickResp to tick")
			continue
		}

		// LiquidityGross = 0 means that the tick is uninitialized
		if tick.LiquidityGross.Cmp(bignumber.ZeroBI) == 0 {
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(Extra{
		Liquidity:                 rpcData.liquidity,
		VolumePerLiquidityInBlock: rpcData.volumePerLiquidityInBlock,
		GlobalState:               rpcData.state,
		FeeConfig:                 rpcData.feeConf,
		Ticks:                     ticks,
		TickSpacing:               int24(rpcData.tickSpacing.Int64()),
		Timepoints:                rpcData.timepoints,
	})

	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.reserve0.String(),
		rpcData.reserve1.String(),
	}

	logger.Infof("[%v] Finish updating state of pool: %v", d.config.DexID, p.Address)

	return p, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		dataStorageOperator common.Address
	)
	res := FetchRPCResult{}

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
		Params: nil,
	}, []interface{}{&res.liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetGlobalState,
		Params: nil,
	}, []interface{}{&res.state})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetDataStorageOperator,
		Params: nil,
	}, []interface{}{&dataStorageOperator})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetTickSpacing,
		Params: nil,
	}, []interface{}{&res.tickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20MethodBalanceOf,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.reserve1})
	}

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process tryAggregate")
		return res, err
	}

	err = d.getPoolFeeConfig(ctx, dataStorageOperator.Hex(), &res.feeConf)
	if err != nil {
		return res, err
	}

	res.timepoints, err = d.getPoolTimepoints(ctx, res.state.TimepointIndex, p.Address)
	if err != nil {
		return res, err
	}

	res.volumePerLiquidityInBlock, err = d.getPoolVolumePerLiquidityInBlock(ctx, common.HexToAddress(p.Address))
	if err != nil {
		return res, err
	}

	return res, err
}

func (d *PoolTracker) getPoolFeeConfig(ctx context.Context, dataStorageOperatorAddress string, feeConf *FeeConfiguration) error {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfig,
		Params: nil,
	}, []interface{}{feeConf})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dataStorageAddress": dataStorageOperatorAddress,
			"error":              err,
		}).Errorf("failed to fetch from data storage operator")
		return err
	}
	return nil
}

func (d *PoolTracker) getPoolTimepoints(ctx context.Context, currentIndex uint16, poolAddress string) (map[uint16]Timepoint, error) {
	timepoints := make(map[uint16]Timepoint, UINT16_MODULO)

	// fetch page by page (backward) until we reach uninitialized or older than 1day
	now := time.Now().Unix()
	yesterday := uint32(now - timepointWindowLimitSeconds)
	// for the 1st page we need to fetch the 2 points after currentIndex, to see if it's the oldest
	end := currentIndex + 3
	// this can underflow (wrap back to end of buffer)
	begin := end - timepointPageSize
	for {
		logger.Debugf("fetching timepoints page %v - %v", begin, end)
		page := make([]TimepointRPC, timepointPageSize)
		rpcRequest := d.ethrpcClient.NewRequest()
		rpcRequest.SetContext(ctx)
		for i := 0; i < timepointPageSize; i += 1 {
			tpIdx := (int64(i) + int64(begin)) % UINT16_MODULO
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    algebraV1PoolABI,
				Target: poolAddress,
				Method: methodGetTimepoints,
				Params: []interface{}{big.NewInt(tpIdx)},
			}, []interface{}{&page[i]})
		}
		_, err := rpcRequest.Aggregate()
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": poolAddress,
				"error":       err,
			}).Errorf("failed to fetch pool timepoints")
			return nil, err
		}

		enough := false
		enoughAtIdx := uint16(0)
		for i, tp := range page {
			needed := true
			tpIdx := uint16(i) + begin
			if !tp.Initialized || tp.BlockTimestamp < yesterday {
				// if this point is too old or not written to yet then might be skipped
				// exception: always save the currentIndex point, it's prev, and it's 2 next points
				if tpIdx == currentIndex || tpIdx == currentIndex-1 || tpIdx == currentIndex+1 || tpIdx == currentIndex+2 {
					needed = true
				} else {
					needed = false
				}
			}
			if needed {
				timepoints[tpIdx] = Timepoint{
					Initialized:                   tp.Initialized,
					BlockTimestamp:                tp.BlockTimestamp,
					TickCumulative:                tp.TickCumulative.Int64(),
					SecondsPerLiquidityCumulative: tp.SecondsPerLiquidityCumulative,
					VolatilityCumulative:          tp.VolatilityCumulative,
					AverageTick:                   int24(tp.AverageTick.Int64()),
					VolumePerLiquidityCumulative:  tp.VolumePerLiquidityCumulative,
				}
			} else {
				enough = true
				enoughAtIdx = tpIdx
			}
		}
		logger.Debugf("done fetching timepoints page %v - %v %v %v", begin, end, enough, enoughAtIdx)

		if enough {
			// fetch the 0th point if it's the oldest and not fetched yet
			// (the oldest point is the 0th if there is no overflow, or the one next to current otherwise)
			_, tp0fetched := timepoints[0]
			tpCurNextIsOldest := timepoints[currentIndex+1].Initialized
			if !tpCurNextIsOldest && !tp0fetched {
				var tp0 TimepointRPC
				rpcRequest.AddCall(&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(0)},
				}, []interface{}{&tp0})

				_, err = rpcRequest.Aggregate()
				if err != nil {
					logger.WithFields(logger.Fields{
						"poolAddress": poolAddress,
						"error":       err,
					}).Errorf("failed to fetch pool timepoints")
					return nil, err
				}

				timepoints[0] = Timepoint{
					Initialized:                   tp0.Initialized,
					BlockTimestamp:                tp0.BlockTimestamp,
					TickCumulative:                tp0.TickCumulative.Int64(),
					SecondsPerLiquidityCumulative: tp0.SecondsPerLiquidityCumulative,
					VolatilityCumulative:          tp0.VolatilityCumulative,
					AverageTick:                   int24(tp0.AverageTick.Int64()),
					VolumePerLiquidityCumulative:  tp0.VolumePerLiquidityCumulative,
				}
			}
			break
		}

		// next page, can be underflow back to end of buffer
		end = begin
		begin = end - timepointPageSize
	}

	return timepoints, nil
}

func (d *PoolTracker) getPoolVolumePerLiquidityInBlock(ctx context.Context, poolAddress common.Address) (*big.Int, error) {
	abiUint256, _ := abi.NewType("uint256", "", nil)
	abi := abi.Arguments{
		// 2 variables are stored in 1 slot, need to read the whole and shift out later
		{Name: "liquidity_volumePerLiquidityInBlock", Type: abiUint256},
	}

	resp, err := d.ethrpcClient.NewRequest().SetContext(ctx).GetStorageAt(
		poolAddress,
		slot3,
		abi,
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"error":       err,
		}).Errorf("failed to fetch pool volumePerLiquidityInBlock")
		return nil, err
	}

	if len(resp) == 1 {
		if bi, ok := resp[0].(*big.Int); ok {
			return new(big.Int).Rsh(bi, 128), nil
		}
	}
	logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"resp":        resp,
	}).Errorf("failed to unmarshal volumePerLiquidityInBlock")
	return nil, ErrUnmarshalVolLiq
}

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	allowSubgraphError := d.config.AllowSubgraphError
	skip := 0
	var ticks []TickResp

	for {
		req := graphql.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks `json:"pool"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Pool == nil {
					logger.WithFields(logger.Fields{
						"poolAddress":        poolAddress,
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Errorf("failed to query subgraph")

					return nil, err
				}
			} else {
				logger.WithFields(logger.Fields{
					"poolAddress":        poolAddress,
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Errorf("failed to query subgraph")

				return nil, err
			}
		}

		if resp.Pool == nil || len(resp.Pool.Ticks) == 0 {
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
