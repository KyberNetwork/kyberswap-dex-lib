package integral

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexType, NewPoolTracker)

type PoolTracker struct {
	algebra.PoolTracker[Timepoint, TimepointRPC]
	config        *Config
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		PoolTracker:   algebra.PoolTracker[Timepoint, TimepointRPC]{EthrpcClient: ethrpcClient},
		config:        cfg,
		graphqlClient: graphqlClient,
	}
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
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.FetchRPCData(ctx, &p, 0)
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
		ExtraTimepoint:   ExtraTimepoint{Timepoints: rpcData.Timepoints},
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

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	res := &FetchRPCResult{}

	req := d.EthrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}

	req.AddCall(&ethrpc.Call{
		ABI:    poolV12ABI,
		Target: p.Address,
		Method: poolLiquidityMethod,
	}, []any{&res.Liquidity})

	rpcState := &GlobalStateFromRPC{}
	req.AddCall(&ethrpc.Call{
		ABI:    poolV12ABI,
		Target: p.Address,
		Method: poolGlobalStateMethod,
	}, []any{rpcState})

	req.AddCall(&ethrpc.Call{
		ABI:    poolV12ABI,
		Target: p.Address,
		Method: poolTickSpacingMethod,
	}, []any{&res.TickSpacing})

	if len(p.Tokens) == 2 {
		req.AddCall(&ethrpc.Call{
			ABI:    abipkg.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abipkg.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve0})

		req.AddCall(&ethrpc.Call{
			ABI:    abipkg.Erc20ABI,
			Target: p.Tokens[1].Address,
			Method: abipkg.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve1})
	}

	var plugin common.Address
	req.AddCall(&ethrpc.Call{
		ABI:    poolV12ABI,
		Target: p.Address,
		Method: poolPluginMethod,
	}, []any{&plugin})

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

	timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, err := d.getPluginData(ctx,
		p, plugin.Hex(), result.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch plugin data")
		return res, err
	}

	res.Timepoints = timepoints
	res.VolatilityOracle = volatilityOracleData
	res.DynamicFee = dynamicFeeData
	res.SlidingFee = slidingFeeData
	res.BlockNumber = result.BlockNumber

	return res, nil
}

func (d *PoolTracker) getPluginData(ctx context.Context, p *entity.Pool, plugin string,
	blockNumber *big.Int) (map[uint16]Timepoint, *VolatilityOraclePlugin, *DynamicFeeConfig, *SlidingFeeConfig, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"plugin":      plugin,
		"dexID":       d.config.DexID,
	})

	req := d.EthrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	volatilityOracleData, volPost := d.getVolatilityOracleData(req, plugin)
	dynamicFeeData, dynPost := d.getDynamicFeeData(req, plugin)
	var slidingFeeData *SlidingFeeConfig
	var sliPost func(resp *ethrpc.Response) error
	if d.config.UseBasePluginV2 {
		slidingFeeData, sliPost = d.getSlidingFeeData(req, plugin)
	}

	resp, err := req.TryAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch plugin data")
		return nil, nil, nil, nil, err
	}
	if volPost(resp) != nil {
		volatilityOracleData = nil // the plugin doesn't use volatility oracle
	}
	if dynPost(resp) != nil {
		dynamicFeeData = nil // the plugin doesn't use dynamic fee
	}
	if sliPost != nil {
		_ = sliPost(resp)
	}

	var timepoints map[uint16]Timepoint
	if volatilityOracleData != nil {
		var extra ExtraTimepoint
		_ = json.Unmarshal([]byte(p.Extra), &extra)
		if timepoints, err = d.getTimepoints(ctx, plugin, blockNumber, volatilityOracleData.TimepointIndex,
			extra.Timepoints); err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch timepoints data from plugin")
			return nil, nil, nil, nil, err
		}
	}

	return timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, nil
}

func (d *PoolTracker) getVolatilityOracleData(req *ethrpc.Request, pluginAddress string) (*VolatilityOraclePlugin,
	func(resp *ethrpc.Response) error) {
	var result VolatilityOraclePlugin
	callsFrom := len(req.Calls)

	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginIsInitializedMethod,
	}, []any{&result.IsInitialized})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginLastTimepointTimestampMethod,
	}, []any{&result.LastTimepointTimestamp})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginTimepointIndexMethod,
	}, []any{&result.TimepointIndex})

	callsTo := len(req.Calls)
	return &result, func(resp *ethrpc.Response) error {
		for i := callsFrom; i < callsTo; i++ {
			if !resp.Result[i] {
				return errors.New("failed to fetch VolatilityOraclePlugin." + req.Calls[i].Method)
			}
		}
		return nil
	}
}

func (d *PoolTracker) getDynamicFeeData(req *ethrpc.Request, pluginAddress string) (*DynamicFeeConfig,
	func(resp *ethrpc.Response) error) {
	var result DynamicFeeConfig
	callsFrom := len(req.Calls)

	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeConfigMethod,
	}, []any{&result})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeZeroToOneMethod,
	}, []any{&result.ZeroToOne})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeOneToZeroMethod,
	}, []any{&result.OneToZero})

	return &result, func(resp *ethrpc.Response) error {
		if !resp.Result[callsFrom] && (!resp.Result[callsFrom+1] || !resp.Result[callsFrom+2]) {
			return errors.New("failed to fetch DynamicFeeConfig")
		}
		return nil
	}
}

func (d *PoolTracker) getSlidingFeeData(req *ethrpc.Request, pluginAddress string) (*SlidingFeeConfig,
	func(resp *ethrpc.Response) error) {
	var cfg SlidingFeeConfig
	var result SlidingFeeConfigRPC
	callsFrom := len(req.Calls)

	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginFeeFactorsMethod,
	}, []any{&result})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginPriceChangeFactorMethod,
	}, []any{&cfg.PriceChangeFactor})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginBaseFeeMethod,
	}, []any{&cfg.BaseFee})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginFeeTypeMethod,
	}, []any{&cfg.FeeType})

	callsTo := len(req.Calls)
	return &cfg, func(resp *ethrpc.Response) error {
		for i := callsFrom; i < callsTo; i++ {
			if !resp.Result[i] {
				return errors.New("failed to fetch SlidingFeeConfig." + req.Calls[i].Method)
			}
		}
		cfg.ZeroToOneFeeFactor = uint256.MustFromBig(result.OneToZeroFeeFactor)
		cfg.OneToZeroFeeFactor = uint256.MustFromBig(result.ZeroToOneFeeFactor)
		return nil
	}
}

func (d *PoolTracker) getTimepoints(ctx context.Context, pluginAddress string, blockNumber *big.Int,
	currentIndex uint16, timepoints map[uint16]Timepoint) (map[uint16]Timepoint, error) {
	return d.GetTimepoints(ctx, &ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginTimepointsMethod,
	}, blockNumber, blockTimestamp()-WINDOW, currentIndex, timepoints)
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

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	_ map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	if len(p.Extra) == 0 {
		return p, nil
	}

	// Extract current ticks from entity pool extra
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return p, err
	}

	if len(extra.Ticks) == 0 {
		return p, nil
	}

	// Use a map here to filter duplicated tick indexes
	seen := make(map[int]struct{}, len(extra.Ticks))
	ticksToRefetch := make([]int, 0, len(extra.Ticks))

	for _, tick := range extra.Ticks {
		if _, exists := seen[tick.Index]; !exists {
			seen[tick.Index] = struct{}{}
			ticksToRefetch = append(ticksToRefetch, tick.Index)
		}
	}

	refetchedTicks, err := t.queryRPCTicksByIndexes(ctx, p.Address, ticksToRefetch, p.BlockNumber)
	if err != nil {
		return p, err
	}

	// convert back to algebra-integral ticks (v3Entities.Tick actually)
	entityPoolTicks := make([]v3Entities.Tick, 0, len(refetchedTicks))
	for _, tick := range refetchedTicks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, v3Entities.Tick{
			Index:          tick.TickIdx,
			LiquidityGross: uint256.MustFromBig(tick.LiquidityGross),
			LiquidityNet:   int256.MustFromBig(tick.LiquidityNet),
		})
	}

	// Sort the ticks by tick index
	if len(entityPoolTicks) > 1 {
		sort.Slice(entityPoolTicks, func(i, j int) bool {
			return entityPoolTicks[i].Index < entityPoolTicks[j].Index
		})
	}

	extra.Ticks = entityPoolTicks
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) newTicksBasedPool(ctx context.Context, p entity.Pool, logs []ethtypes.Log) (tickspkg.TicksBasedPool,
	error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to transform entity pool to ticks based pool")
		return ticksBasedPool, err
	}

	ticks, err := t.fetchTicksFromLogs(ctx, p.Address, logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to FetchTicksFromLogs")
		return ticksBasedPool, err
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)
	ticksBasedPool.BlockNumber = blockNumber

	if len(ticks) == 0 {
		return ticksBasedPool, nil
	}

	if err := tickspkg.ValidatePoolTicks(ticksBasedPool, ticks); err != nil {
		l.WithFields(logger.Fields{
			"numTicks": len(ticks),
			"error":    err,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		l.WithFields(logger.Fields{
			"numTicks": len(ticksBasedPool.Ticks),
		}).Info("fetch all ticks for pool")

		ticks, err = t.fetchAllTicksForPool(ctx, ticksBasedPool, ticks)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch all ticks")

			return ticksBasedPool, err
		}

		if err := tickspkg.ValidateAllPoolTicks(ticksBasedPool, ticks); err != nil {
			l.WithFields(logger.Fields{
				"numTicks": len(ticks),
				"error":    err,
			}).Warnf("invalid pool ticks data after fetching all ticks stored in pool")
		}
	}

	for _, tick := range ticks {
		ticksBasedPool.Ticks[tick.TickIdx] = tick
	}

	return ticksBasedPool, nil
}

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	blockNumber := ticksBasedPool.BlockNumber

	rpcState, err := t.FetchRPCData(ctx, &p, blockNumber)
	if err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			rpcState, err = t.FetchRPCData(ctx, &p, 0)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to fetch latest state from RPC")
				return p, err
			}
		} else {
			l.WithFields(logger.Fields{
				"error":       err,
				"blockNumber": blockNumber,
			}).Error("failed to fetch state from RPC")
			return p, err
		}
	}

	entityPoolTicks := make([]v3Entities.Tick, 0, len(ticksBasedPool.Ticks))
	for _, tick := range ticksBasedPool.Ticks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, v3Entities.Tick{
			Index:          tick.TickIdx,
			LiquidityGross: uint256.MustFromBig(tick.LiquidityGross),
			LiquidityNet:   int256.MustFromBig(tick.LiquidityNet),
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

	extraBytes, err := json.Marshal(Extra{
		Liquidity:        uint256.MustFromBig(rpcState.Liquidity),
		GlobalState:      rpcState.State,
		Ticks:            entityPoolTicks,
		TickSpacing:      int32(rpcState.TickSpacing.Int64()),
		ExtraTimepoint:   ExtraTimepoint{Timepoints: rpcState.Timepoints},
		VolatilityOracle: rpcState.VolatilityOracle,
		DynamicFee:       rpcState.DynamicFee,
		SlidingFee:       rpcState.SlidingFee,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcState.Reserve0.String(),
		rpcState.Reserve1.String(),
	}

	return p, nil
}

func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context, address string, logs []ethtypes.Log,
) ([]tickspkg.Tick, error) {
	if len(logs) == 0 {
		return nil, nil
	}

	l := logger.WithFields(logger.Fields{
		"address": address,
	})

	tickIndexes, err := t.getTickIndexesFromLogs(logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to getTickIndexesFromEvents")
		return nil, err
	}

	if len(tickIndexes) == 0 {
		return nil, nil
	}
	blockNumber := eth.GetBlockNumberFromLogs(logs)

	return t.queryRPCTicksByIndexes(ctx, address, tickIndexes, blockNumber)
}

func (t *PoolTracker) fetchAllTicksForPool(
	ctx context.Context,
	pool tickspkg.TicksBasedPool,
	ticksFromLogs []tickspkg.Tick,
) ([]tickspkg.Tick, error) {
	isTickFromLogs := map[int]struct{}{}
	lo.ForEach(ticksFromLogs, func(item tickspkg.Tick, index int) {
		isTickFromLogs[item.TickIdx] = struct{}{}
	})

	tickIdsFromPool := make([]int, 0, len(pool.Ticks))
	for tickIdx := range pool.Ticks {
		if _, ok := isTickFromLogs[tickIdx]; !ok {
			tickIdsFromPool = append(tickIdsFromPool, tickIdx)
		}
	}

	ticksFromPool, err := t.queryRPCTicksByIndexes(ctx, pool.Address, tickIdsFromPool, pool.BlockNumber)
	if err != nil {
		return nil, err
	}

	ticksMap := make(map[int]tickspkg.Tick)
	for _, tick := range ticksFromPool {
		ticksMap[tick.TickIdx] = tick
	}
	for _, tick := range ticksFromLogs {
		ticksMap[tick.TickIdx] = tick
	}

	return lo.Values(ticksMap), nil
}

// getTickIndexesFromLogs returns all tick indexes from logs.
func (t *PoolTracker) getTickIndexesFromLogs(logs []ethtypes.Log) ([]int, error) {
	tickSet := make(map[int]struct{}, len(logs)*2)

	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		switch event.Topics[0] {
		case poolV12ABI.Events["Mint"].ID: // note : There's no difference between v1.0 and v1.2
			mint, err := poolV12Filterer.ParseMint(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse mint event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   mint,
			}).Debug("decode mint event")

			tickSet[int(mint.BottomTick.Int64())] = struct{}{}
			tickSet[int(mint.TopTick.Int64())] = struct{}{}

		case poolV12ABI.Events["Burn"].ID:
			burn, err := poolV12Filterer.ParseBurn(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse burn event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   burn,
			}).Debug("decode burn event")

			tickSet[int(burn.BottomTick.Int64())] = struct{}{}
			tickSet[int(burn.TopTick.Int64())] = struct{}{}

		case poolV10ABI.Events["Burn"].ID:
			burn, err := poolV10Filterer.ParseBurn(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse burn event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   burn,
			}).Debug("decode burn event")

			tickSet[int(burn.BottomTick.Int64())] = struct{}{}
			tickSet[int(burn.TopTick.Int64())] = struct{}{}

		default:
			metrics.IncrUnprocessedEventTopic(pooltypes.PoolTypes.AlgebraIntegral, event.Topics[0].Hex())
		}
	}

	ticks := make([]int, 0, len(tickSet))
	for tick := range tickSet {
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

func (t *PoolTracker) queryRPCTicksByIndexes(
	ctx context.Context, address string, tickIndexes []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		return t.queryRPCTicksByChunk(ctx, address, tickIndexes, blockNumber)
	}

	totalTicks := len(tickIndexes)
	ticks := make([]tickspkg.Tick, 0, totalTicks)
	for i := 0; i < totalTicks; i += tickChunkSize {
		toIdx := i + tickChunkSize
		if toIdx > totalTicks {
			toIdx = totalTicks
		}

		newTicks, err := t.queryRPCTicksByChunk(ctx, address, tickIndexes[i:toIdx], blockNumber)
		if err != nil {
			return nil, err
		}

		ticks = append(ticks, newTicks...)
	}

	return ticks, nil
}

// queryRPCTicksByChunk returns univ3 Ticks data.
func (t *PoolTracker) queryRPCTicksByChunk(
	ctx context.Context, addr string, ticks []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	tickResponses := make([]TicksResp, len(ticks))
	ticksRequest := t.EthrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		ticksRequest.SetBlockNumber(&blockNumberBI)
	}

	for id, tick := range ticks {
		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    poolV12ABI,
			Target: addr,
			Method: poolTicksMethod,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	l := logger.WithFields(logger.Fields{
		"address": addr,
	})

	l.WithFields(logger.Fields{
		"len":   len(ticksRequest.Calls),
		"ticks": ticks,
	}).Debug("fetching ticks")

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksByChunk(ctx, addr, ticks, 0)
		}

		logger.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process aggregate to get ticks")
		return nil, err
	}

	result := make([]tickspkg.Tick, len(ticks))
	for id, tickResponse := range tickResponses {
		result[id] = tickspkg.Tick{
			TickIdx:        ticks[id],
			LiquidityGross: tickResponse.LiquidityTotal,
			LiquidityNet:   tickResponse.LiquidityDelta,
		}
	}

	return result, nil
}
