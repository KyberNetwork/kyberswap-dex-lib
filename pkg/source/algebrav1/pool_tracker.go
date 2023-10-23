package algebrav1

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
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
		Liquidity:   rpcData.liquidity,
		GlobalState: rpcData.state,
		Ticks:       ticks,
		TickSpacing: int24(rpcData.tickSpacing.Int64()),
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

	logger.Infof("[%v] Finish updating state of pool: %v, approximate fee %v %v", d.config.DexID, p.Address, rpcData.state.FeeZto, rpcData.state.FeeOtz)

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

	// the globalstate abi are slightly different across versions
	var rpcState interface{}
	if d.config.UseDirectionalFee {
		rpcState = &rpcGlobalStateDirFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1DirFeePoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
			Params: nil,
		}, []interface{}{rpcState})
	} else {
		rpcState = &rpcGlobalStateSingleFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1PoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
			Params: nil,
		}, []interface{}{rpcState})
	}

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

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to process tryAggregate")
		return res, err
	}

	if d.config.UseDirectionalFee {
		rpcStateRes := rpcState.(*rpcGlobalStateDirFee)
		res.state = GlobalState{
			Price:              rpcStateRes.Price,
			Tick:               rpcStateRes.Tick,
			FeeZto:             rpcStateRes.FeeZto,
			FeeOtz:             rpcStateRes.FeeOtz,
			TimepointIndex:     rpcStateRes.TimepointIndex,
			CommunityFeeToken0: uint16(rpcStateRes.CommunityFeeToken0),
			CommunityFeeToken1: uint16(rpcStateRes.CommunityFeeToken1),
			Unlocked:           rpcStateRes.Unlocked,
		}
	} else {
		// for v1 without directional fee, we'll use Fee for both FeeZto/FeeOtz
		rpcStateRes := rpcState.(*rpcGlobalStateSingleFee)
		res.state = GlobalState{
			Price:              rpcStateRes.Price,
			Tick:               rpcStateRes.Tick,
			FeeZto:             rpcStateRes.Fee,
			FeeOtz:             rpcStateRes.Fee,
			TimepointIndex:     rpcStateRes.TimepointIndex,
			CommunityFeeToken0: rpcStateRes.CommunityFeeToken0,
			CommunityFeeToken1: rpcStateRes.CommunityFeeToken1,
			Unlocked:           rpcStateRes.Unlocked,
		}
	}

	if !d.config.SkipFeeCalculating {
		err = d.approximateFee(ctx, p.Address, dataStorageOperator.Hex(), &res.state, res.liquidity)
		if err != nil {
			return res, err
		}
	}

	if !res.state.Unlocked {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Info("pool has been locked and not usable")
	}

	return res, err
}

func (d *PoolTracker) approximateFee(ctx context.Context, poolAddress, dataStorageOperator string, state *GlobalState, currentLiquidity *big.Int) error {
	// fee approximation: assume that the swap will be soon after this
	blockTimestamp := uint32(time.Now().Unix())
	yesterday := blockTimestamp - WINDOW
	timepoints, err := d.getPoolTimepoints(ctx, state.TimepointIndex, poolAddress, yesterday)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"error":       err,
		}).Error("failed to fetch pool timepoints")
		return err
	}

	if timepoints == nil {
		// not initialized pool has been locked already, but set here just for sure
		state.Unlocked = false
		return nil
	}

	feeConf := FeeConfiguration{}
	feeConfZto := FeeConfiguration{}
	feeConfOtz := FeeConfiguration{}
	if d.config.UseDirectionalFee {
		err = d.getPoolDirectionalFeeConfig(ctx, dataStorageOperator, &feeConfZto, &feeConfOtz)
	} else {
		err = d.getPoolFeeConfig(ctx, dataStorageOperator, &feeConf)
	}
	if err != nil {
		return err
	}

	volumePerLiquidityInBlock, err := d.getPoolVolumePerLiquidityInBlock(ctx, common.HexToAddress(poolAddress))
	if err != nil {
		return err
	}

	ts := TimepointStorage{
		data:    timepoints,
		updates: map[uint16]Timepoint{},
	}
	currentTick := int24(state.Tick.Int64())
	newTimepointIndex, err := ts.write(
		state.TimepointIndex,
		blockTimestamp,
		currentTick,
		currentLiquidity,
		volumePerLiquidityInBlock,
	)
	if err != nil {
		return err
	}

	if d.config.UseDirectionalFee {
		state.FeeZto, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConfZto)
		if err != nil {
			return err
		}
		state.FeeOtz, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConfOtz)
		if err != nil {
			return err
		}
	} else {
		state.FeeZto, err = ts._getNewFee(blockTimestamp, currentTick, newTimepointIndex, currentLiquidity, &feeConf)
		if err != nil {
			return err
		}
		state.FeeOtz = state.FeeZto
	}
	return nil
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

func (d *PoolTracker) getPoolDirectionalFeeConfig(ctx context.Context, dataStorageOperatorAddress string, feeConfZto *FeeConfiguration, feeConfOtz *FeeConfiguration) error {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigZto,
		Params: nil,
	}, []interface{}{feeConfZto},
	).AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorAPI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigOtz,
		Params: nil,
	}, []interface{}{feeConfOtz})

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

func (d *PoolTracker) getPoolTimepoints(ctx context.Context, currentIndex uint16, poolAddress string, yesterday uint32) (map[uint16]Timepoint, error) {
	timepoints := make(map[uint16]Timepoint, UINT16_MODULO)

	currentIndexPrev := currentIndex - 1
	currentIndexNext := currentIndex + 1
	currentIndexNextNext := currentIndex + 2

	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	rpcRequest.Calls = make([]*ethrpc.Call, 0, timepointPageSize)
	page := make([]TimepointRPC, timepointPageSize)

	// fetch page by page (backward) until we reach uninitialized or older than 1day
	end := currentIndex + 1
	// this can underflow (wrap back to end of buffer)
	begin := end - timepointPageSize
	for {
		logger.Debugf("fetching timepoints page %v - %v", begin, end)

		rpcRequest.Calls = rpcRequest.Calls[:0]
		for i := uint16(0); i < timepointPageSize; i += 1 {
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
			tpIdx := uint16(i) + begin
			if !tp.Initialized || tp.BlockTimestamp < yesterday {
				// if this point is too old or not written to yet then skipped
				// TODO: check if we've wrapped full circle yet
				enough = true
				enoughAtIdx = tpIdx
			} else {
				timepoints[tpIdx] = tp.toTimepoint()
			}
		}
		logger.Debugf("done fetching timepoints page %v - %v %v %v", begin, end, enough, enoughAtIdx)

		if enough {
			// fetch some additional timepoints
			// (some of them might already been fetched but still refetch anyway for simplicity)
			var tp0, tpCurNext, tpCurNextNext, tpLowest, tpCurPrev TimepointRPC
			rpcRequest.Calls = rpcRequest.Calls[:0]
			rpcRequest.AddCall(
				&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(0)},
				},
				[]interface{}{&tp0},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(int64(currentIndexNext))},
				},
				[]interface{}{&tpCurNext},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(int64(currentIndexNextNext))},
				},
				[]interface{}{&tpCurNextNext},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(int64(enoughAtIdx))},
				},
				[]interface{}{&tpLowest},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraV1PoolABI,
					Target: poolAddress,
					Method: methodGetTimepoints,
					Params: []interface{}{big.NewInt(int64(currentIndexPrev))},
				},
				[]interface{}{&tpCurPrev},
			)

			_, err = rpcRequest.Aggregate()
			if err != nil {
				logger.WithFields(logger.Fields{
					"poolAddress": poolAddress,
					"error":       err,
				}).Errorf("failed to fetch pool timepoints")
				return nil, err
			}

			timepoints[0] = tp0.toTimepoint()
			timepoints[currentIndexNext] = tpCurNext.toTimepoint()
			timepoints[currentIndexNextNext] = tpCurNextNext.toTimepoint()
			timepoints[enoughAtIdx] = tpLowest.toTimepoint() // needed to ensure binary search will terminate
			timepoints[currentIndexPrev] = tpCurPrev.toTimepoint()

			break
		}

		// next page, can be underflow back to end of buffer
		end = begin
		begin = end - timepointPageSize
		if begin <= currentIndex && currentIndex < end {
			//we've wrapped around full circle, so break here
			break
		}
	}

	// the currentIndex might has been increased onchain while we're fetching
	// so detect staleness here
	currentTs := timepoints[currentIndex].BlockTimestamp
	if timepoints[currentIndexNext].Initialized && timepoints[currentIndexNext].BlockTimestamp > currentTs {
		return nil, ErrStaleTimepoints
	}
	if timepoints[currentIndexNextNext].Initialized && timepoints[currentIndexNextNext].BlockTimestamp > currentTs {
		return nil, ErrStaleTimepoints
	}

	if !timepoints[currentIndex].Initialized {
		// some new pools don't have timepoints initialized yet, ignore them
		return nil, nil
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
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
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
		resp.Meta.CheckIsLagging(d.config.DexID, poolAddress)

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
