package integral

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	l.Info("Start getting new state of pool")

	var (
		rpcData   FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.fetchRPCData(ctx, p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")
		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		if d.config.AlwaysUseTickLens {
			poolTicks, err = d.getPoolTicksFromSC(ctx, p, param)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

		poolTicks, err = d.getPoolTicks(ctx, p.Address)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to query subgraph for pool ticks")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool state")
		return entity.Pool{}, err
	}

	ticks := make([]v3Entities.Tick, 0, len(poolTicks))
	for _, tickResp := range poolTicks {
		tick, err := tickResp.transformTickRespToTick()
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			continue
		}

		// LiquidityGross = 0 means that the tick is uninitialized
		if tick.LiquidityGross.IsZero() {
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(&Extra{
		Liquidity:        uint256.MustFromBig(rpcData.Liquidity),
		GlobalState:      rpcData.State,
		Ticks:            ticks,
		TickSpacing:      int32(rpcData.TickSpacing.Int64()),
		Timepoints:       rpcData.Timepoints,
		VolatilityOracle: rpcData.VolatilityOracle,
		SlidingFee:       rpcData.SlidingFee,
		DynamicFee:       rpcData.DynamicFee,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}

	var blockNumber uint64
	if rpcData.BlockNumber != nil {
		blockNumber = rpcData.BlockNumber.Uint64()
	}
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"lastFee": rpcData.State.LastFee,
	}).Info("Finish updating state of pool")

	return p, nil
}

func (d *PoolTracker) FetchStateFromRPC(ctx context.Context, p entity.Pool, blockNumber uint64) ([]byte, error) {
	rpcData, err := d.fetchRPCData(ctx, p, blockNumber)
	if err != nil {
		return nil, err
	}

	rpcDataBytes, err := json.Marshal(rpcData)
	if err != nil {
		return nil, err
	}

	return rpcDataBytes, nil
}

func (d *PoolTracker) fetchRPCData(ctx context.Context, p entity.Pool, blockNumber uint64) (FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	res := FetchRPCResult{}

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}

	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolLiquidityMethod,
		Params: nil,
	}, []interface{}{&res.Liquidity})

	rpcState := &GlobalStateFromRPC{}
	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolGlobalStateMethod,
		Params: nil,
	}, []interface{}{rpcState})

	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolTickSpacingMethod,
		Params: nil,
	}, []interface{}{&res.TickSpacing})

	if len(p.Tokens) == 2 {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20BalanceOfMethod,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.Reserve0})

		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20BalanceOfMethod,
			Params: []interface{}{common.HexToAddress(p.Address)},
		}, []interface{}{&res.Reserve1})
	}

	result, err := req.Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool data")
		return res, err
	}

	res.State = GlobalState{
		Price:        uint256.MustFromBig(rpcState.Price),
		Tick:         int32(rpcState.Tick.Int64()),
		LastFee:      rpcState.LastFee,
		PluginConfig: rpcState.PluginConfig,
		CommunityFee: rpcState.CommunityFee,
		Unlocked:     rpcState.Unlocked,
	}

	timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, err := d.getPluginData(ctx, p.Address,
		result.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch plugin data")
		return res, err
	}

	res.Timepoints = timepoints
	res.VolatilityOracle = volatilityOracleData
	res.SlidingFee = slidingFeeData
	res.DynamicFee = dynamicFeeData
	res.BlockNumber = result.BlockNumber

	return res, nil
}

func (d *PoolTracker) getPluginData(ctx context.Context, poolAddress string, blockNumber *big.Int) (
	map[uint16]Timepoint, VolatilityOraclePlugin, DynamicFeeConfig, SlidingFeeConfig, error) {

	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	var plugin common.Address

	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: poolAddress,
		Method: poolPluginMethod,
		Params: nil,
	}, []interface{}{&plugin})

	_, err := req.Call()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch Plugin address from pool")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	volatilityOracleData, err := d.getVolatilityOracleData(ctx, plugin.Hex(), blockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch VolatilityOracle data from plugin")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	timepoints, err := d.fetchTimepoints(ctx, plugin.Hex(), blockNumber, volatilityOracleData.TimepointIndex)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch timepoints data from plugin")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	dynamicFeeData, err := d.getDynamicFeeData(ctx, plugin.Hex(), blockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch DynamicFee data from plugin")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	var slidingFeeData SlidingFeeConfig
	if d.config.UseBasePluginV2 {
		slidingFeeData, err = d.getSlidingFeeData(ctx, plugin.Hex(), blockNumber)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch Sliding data from plugin")
			return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
		}
	}

	return timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, nil
}

func (d *PoolTracker) getVolatilityOracleData(ctx context.Context, pluginAddress string,
	blockNumber *big.Int) (VolatilityOraclePlugin, error) {
	var result VolatilityOraclePlugin

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginIsInitializedMethod,
		Params: nil,
	}, []interface{}{&result.IsInitialized})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginLastTimepointTimestampMethod,
		Params: nil,
	}, []interface{}{&result.LastTimepointTimestamp})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginTimepointIndexMethod,
		Params: nil,
	}, []interface{}{&result.TimepointIndex})

	_, err := req.Aggregate()
	if err != nil {
		return VolatilityOraclePlugin{}, err
	}

	return result, nil
}

func (d *PoolTracker) getSlidingFeeData(ctx context.Context, pluginAddress string,
	blockNumber *big.Int) (SlidingFeeConfig, error) {
	var result SlidingFeeConfigRPC

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginFeeFactorsMethod,
		Params: nil,
	}, []interface{}{&result})

	_, err := req.Call()
	if err != nil {
		return SlidingFeeConfig{}, err
	}

	return SlidingFeeConfig{
		ZeroToOneFeeFactor: uint256.MustFromBig(result.OneToZeroFeeFactor),
		OneToZeroFeeFactor: uint256.MustFromBig(result.ZeroToOneFeeFactor),
	}, nil
}

func (d *PoolTracker) getDynamicFeeData(ctx context.Context, pluginAddress string,
	blockNumber *big.Int) (DynamicFeeConfig, error) {
	var result DynamicFeeConfig

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeConfigMethod,
		Params: nil,
	}, []interface{}{&result})

	_, err := req.Call()
	if err != nil {
		return DynamicFeeConfig{}, err
	}

	return result, nil
}

func (d *PoolTracker) fetchTimepoints(ctx context.Context, pluginAddress string, blockNumber *big.Int,
	currentIndex uint16) (map[uint16]Timepoint, error) {
	blockTimestamp := uint32(time.Now().Unix())
	yesterday := blockTimestamp - WINDOW
	timepoints, err := d.getPoolTimepoints(ctx, blockNumber, pluginAddress, currentIndex, yesterday)
	if err != nil {
		return nil, err
	}

	return timepoints, nil
}

func (d *PoolTracker) getPoolTimepoints(ctx context.Context, blockNumber *big.Int, pluginAddress string,
	currentIndex uint16, yesterday uint32) (map[uint16]Timepoint, error) {
	timepoints := make(map[uint16]Timepoint, UINT16_MODULO)

	currentIndexPrev := currentIndex - 1
	currentIndexNext := currentIndex + 1
	currentIndexNextNext := currentIndex + 2

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	req.Calls = make([]*ethrpc.Call, 0, timepointPageSize)
	page := make([]TimepointRPC, timepointPageSize)

	// fetch page by page (backward) until we reach uninitialized or older than 1day
	end := currentIndex + 1
	// this can underflow (wrap back to end of buffer)
	begin := end - timepointPageSize
	for {
		logger.Debugf("fetching timepoints page %v - %v", begin, end)

		req.Calls = req.Calls[:0]
		for i := uint16(0); i < timepointPageSize; i += 1 {
			tpIdx := (int64(i) + int64(begin)) % UINT16_MODULO
			req.AddCall(&ethrpc.Call{
				ABI:    algebraBasePluginV2ABI,
				Target: pluginAddress,
				Method: votalityOraclePluginTimepointsMethod,
				Params: []interface{}{big.NewInt(tpIdx)},
			}, []interface{}{&page[i]})
		}
		_, err := req.Aggregate()
		if err != nil {
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
			req.Calls = req.Calls[:0]
			req.AddCall(
				&ethrpc.Call{
					ABI:    algebraBasePluginV2ABI,
					Target: pluginAddress,
					Method: votalityOraclePluginTimepointsMethod,
					Params: []interface{}{big.NewInt(0)},
				},
				[]interface{}{&tp0},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraBasePluginV2ABI,
					Target: pluginAddress,
					Method: votalityOraclePluginTimepointsMethod,
					Params: []interface{}{big.NewInt(int64(currentIndexNext))},
				},
				[]interface{}{&tpCurNext},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraBasePluginV2ABI,
					Target: pluginAddress,
					Method: votalityOraclePluginTimepointsMethod,
					Params: []interface{}{big.NewInt(int64(currentIndexNextNext))},
				},
				[]interface{}{&tpCurNextNext},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraBasePluginV2ABI,
					Target: pluginAddress,
					Method: votalityOraclePluginTimepointsMethod,
					Params: []interface{}{big.NewInt(int64(enoughAtIdx))},
				},
				[]interface{}{&tpLowest},
			).AddCall(
				&ethrpc.Call{
					ABI:    algebraBasePluginV2ABI,
					Target: pluginAddress,
					Method: votalityOraclePluginTimepointsMethod,
					Params: []interface{}{big.NewInt(int64(currentIndexPrev))},
				},
				[]interface{}{&tpCurPrev},
			)

			_, err = req.Aggregate()
			if err != nil {
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
			// we've wrapped around full circle, so break here
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

func (d *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       d.config.DexID,
	})

	allowSubgraphError := d.config.AllowSubgraphError
	skip := 0
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := d.graphqlClient.Run(ctx, req, &resp); err != nil {
			if allowSubgraphError {
				if resp.Pool == nil {
					l.WithFields(logger.Fields{
						"error":              err,
						"allowSubgraphError": allowSubgraphError,
					}).Error("failed to query subgraph")

					return nil, err
				}
			} else {
				l.WithFields(logger.Fields{
					"error":              err,
					"allowSubgraphError": allowSubgraphError,
				}).Error("failed to query subgraph")

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
