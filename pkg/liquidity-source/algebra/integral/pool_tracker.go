package integral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)

type PoolTracker struct {
	algebra.PoolTracker[Timepoint, TimepointRPC]
	config        *Config
	graphqlClient *graphqlpkg.Client
	lensGuard     *ephemeral.Guard
	// pluginLensCall runs AlgebraIntegralPluginLens, callPluginLens but for tests.
	pluginLensCall func(ctx context.Context, block *big.Int, plugin common.Address, calls [][]byte,
		walk pluginWalk) (pluginSnapshot, error)
	// readTicks reads ticks by index at a block, queryRPCTicksByIndexes but for tests.
	readTicks func(ctx context.Context, address string, tickIndexes []int, blockNumber uint64,
		overrides map[common.Address]gethclient.OverrideAccount) ([]tickspkg.Tick, error)
}

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	t := &PoolTracker{
		EthrpcClient:  ethrpcClient,
		config:        cfg,
		graphqlClient: graphqlClient,
		lensGuard:     ephemeral.NewLensGuard(cfg.LensGuard, cfg.LensBatch),
	}
	t.pluginLensCall = t.callPluginLens
	t.readTicks = t.queryRPCTicksByIndexes
	return t
}

func (d *PoolTracker) BootstrapPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	l.Info("Start getting new state of pool")

	if d.config.UseLens {
		p, _, err := d.bootstrapFromLens(ctx, p, param.Logs, nil, l)
		return p, err
	}

	// The state is read at a block named first, which the pool is dated by: the caller reads the
	// ticks of a bootstrapped pool again at its block, and the pool's older one, its creation block
	// for a pool its factory made, drops every tick created since. A multicall's own block is not
	// the chain's number on every chain, so it is not the one taken.
	head, err := d.EthrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get the head block")
		return entity.Pool{}, err
	}

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = d.FetchRPCData(ctx, &p, head, nil)
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

	return withState(p, rpcData, ticks, param.Logs, head, l)
}

var _ sourcePool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)

// NextUpdateAt asks for a pool still without state, its lens read declined by the guard, to be
// updated again as soon as the guard would take it: it may see no log for a long time, and only an
// update gives it its state. Every other pool needs no update without an event.
func (d *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	if d.config.UseLens && !hasState(p) {
		return d.lensGuard.ReadyAt(p.Address, time.Now()).Unix()
	}
	return 0
}

var _ sourcePool.IPoolTrackerWithReset = (*PoolTracker)(nil)

// ResetPoolState drops the pool's state for its next update to read it whole through the lens, at
// a block no lower than block. Without the lens nothing can read it whole, and the pool is kept.
func (d *PoolTracker) ResetPoolState(p entity.Pool, block uint64) (entity.Pool, bool) {
	if !d.config.UseLens {
		return p, false
	}
	return withoutState(p, block), true
}

// bootstrapFromLens reads the pool's state and ticks through the lens at block (nil for the
// latest), then its plugin state and reserves, which the lens does not return, at the same block.
// It returns the pool with the block the lens read it at.
func (d *PoolTracker) bootstrapFromLens(ctx context.Context, p entity.Pool, logs []ethtypes.Log, block *big.Int,
	l logger.Logger) (entity.Pool, uint64, error) {
	snap, err := d.snapshot(ctx, p.Address, block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to read pool through the lens")
		return entity.Pool{}, 0, err
	}

	rpcData, err := d.FetchRPCData(ctx, &p, snap.block, nil)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch data from RPC")
		return entity.Pool{}, 0, err
	}
	if rpcData.Liquidity.Cmp(snap.liquidity) != 0 || int(rpcData.State.Tick) != snap.tick ||
		rpcData.State.Price.ToBig().Cmp(snap.sqrtPriceX96) != 0 {
		return entity.Pool{}, 0, fmt.Errorf("pool %s at block %d: RPC and lens disagree on price, tick or liquidity",
			p.Address, snap.block)
	}

	p, err = withState(p, rpcData, toEntityTicks(snap.ticks), logs, snap.block, l)
	return p, snap.block, err
}

// withState writes the fetched state into p, read at block.
func withState(p entity.Pool, rpcData *FetchRPCResult, ticks []v3Entities.Tick, logs []ethtypes.Log,
	block uint64, l logger.Logger) (entity.Pool, error) {
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
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(logs).BlockTimestamp))
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = max(p.BlockNumber, block, lo.LastOrEmpty(logs).BlockNumber)

	l.WithFields(logger.Fields{
		"lastFee": rpcData.State.LastFee,
	}).Info("Finish updating state of pool")

	return p, nil
}

// toEntityTicks turns a tick set into the Extra's shape, ascending, without the ticks holding no
// liquidity.
func toEntityTicks(ticks []tickspkg.Tick) []v3Entities.Tick {
	out := make([]v3Entities.Tick, 0, len(ticks))
	for _, tick := range ticks {
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}
		out = append(out, v3Entities.Tick{
			Index:          tick.TickIdx,
			LiquidityGross: uint256.MustFromBig(tick.LiquidityGross),
			LiquidityNet:   int256.MustFromBig(tick.LiquidityNet),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

func (d *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64,
	overrides map[common.Address]gethclient.OverrideAccount) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       d.config.DexID,
	})

	res := &FetchRPCResult{}

	req := d.EthrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	if blockNumber > 0 {
		req.SetBlockNumber(big.NewInt(int64(blockNumber)))
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

	timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, err := d.getPluginData(ctx, p, plugin,
		result.BlockNumber, overrides)
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

func (d *PoolTracker) getPluginData(ctx context.Context, p *entity.Pool, pluginAddr common.Address,
	blockNumber *big.Int, overrides map[common.Address]gethclient.OverrideAccount) (map[uint16]Timepoint,
	*VolatilityOraclePlugin, *DynamicFeeConfig, *SlidingFeeConfig, error) {
	if pluginAddr == valueobject.AddrZero {
		return nil, nil, nil, nil, nil
	}
	plugin := hexutil.Encode(pluginAddr[:])
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"plugin":      plugin,
		"dexID":       d.config.DexID,
	})

	// The lens reads the plugin in one call, but a state override applies only to a multicall.
	if d.config.UseLens && overrides == nil {
		timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, err := d.readPluginData(p, plugin,
			newPluginLensRead(ctx, d, pluginAddr, blockNumber))
		if err == nil {
			return timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, nil
		}
		l.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to read the plugin through the lens, reading it by multicall")
	}

	timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, err := d.readPluginData(p, plugin,
		rpcPluginRead{d: d, ctx: ctx, plugin: plugin, block: blockNumber, overrides: overrides})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch plugin data")
		return nil, nil, nil, nil, err
	}
	return timepoints, volatilityOracleData, dynamicFeeData, slidingFeeData, nil
}

// readPluginData reads the plugin's settings, then, when it has a volatility oracle, its timepoints
// from those p holds, making the reads through read.
func (d *PoolTracker) readPluginData(p *entity.Pool, plugin string, read pluginRead) (map[uint16]Timepoint,
	*VolatilityOraclePlugin, *DynamicFeeConfig, *SlidingFeeConfig, error) {
	req := d.EthrpcClient.NewRequest()

	volatilityOracleData, volPost := d.getVolatilityOracleData(req, plugin)
	dynamicFeeData, dynPost := d.getDynamicFeeData(req, plugin)
	var slidingFeeData *SlidingFeeConfig
	var sliPost func(resp *ethrpc.Response) error
	if d.config.UseBasePluginV2 {
		slidingFeeData, sliPost = d.getSlidingFeeData(req, plugin)
	}

	var extra ExtraTimepoint
	_ = json.Unmarshal([]byte(p.Extra), &extra)

	resp, err := read.settings(req, extra.Timepoints)
	if err != nil {
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
		if timepoints, err = read.timepoints(extra.Timepoints, volatilityOracleData.TimepointIndex,
			volatilityOracleData.LastTimepointTimestamp); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("timepoints: %w", err)
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
	// The calls above must succeed. The plugins have one of the calls below: feeType (Thena) or the two switches
	// (Camelot, Cypher, Sheriff).
	feeTypeCall := len(req.Calls)
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginFeeTypeMethod,
	}, []any{&cfg.FeeType})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginDynamicFeeEnabledMethod,
	}, []any{&cfg.DynamicFeeEnabled})
	req.AddCall(&ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginSlidingFeeEnabledMethod,
	}, []any{&cfg.SlidingFeeEnabled})

	return &cfg, func(resp *ethrpc.Response) error {
		for i := callsFrom; i < feeTypeCall; i++ {
			if !resp.Result[i] {
				return errors.New("failed to fetch SlidingFeeConfig." + req.Calls[i].Method)
			}
		}
		// s_feeFactors is (zeroToOneFeeFactor, oneToZeroFeeFactor), as the pool's own swaps confirm.
		zeroToOne, oneToZero := result.ZeroToOneFeeFactor, result.OneToZeroFeeFactor
		if resp.Result[feeTypeCall] {
			// Thena's plugin, read the other way round since it was added; no Thena pool runs the sliding fee
			// to check it against.
			zeroToOne, oneToZero = oneToZero, zeroToOne
		}
		cfg.ZeroToOneFeeFactor = uint256.MustFromBig(zeroToOne)
		cfg.OneToZeroFeeFactor = uint256.MustFromBig(oneToZero)
		return nil
	}
}

func (d *PoolTracker) getTimepoints(ctx context.Context, pluginAddress string, blockNumber *big.Int,
	currentIndex uint16, currentTimestamp uint32, timepoints map[uint16]Timepoint,
	overrides map[common.Address]gethclient.OverrideAccount) (map[uint16]Timepoint, error) {
	return d.GetTimepoints(ctx, &ethrpc.Call{
		ABI:    basePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginTimepointsMethod,
	}, blockNumber, blockTimestamp()-WINDOW, currentIndex, currentTimestamp, timepoints, overrides)
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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, param sourcePool.GetNewPoolStateParams) (entity.Pool, error) {
	if t.config.UseLens && !hasState(p) {
		l := logger.WithFields(logger.Fields{
			"address":  p.Address,
			"exchange": p.Exchange,
		})
		return tickspkg.StateFromLens(ctx, t.lensGuard, p, param,
			func(ctx context.Context, block *big.Int) (entity.Pool, uint64, error) {
				return t.bootstrapFromLens(ctx, p, param.Logs, block, l)
			},
			l), nil
	}

	return t.getNewPoolState(ctx, p, param.Logs, param.BlockHeaders, nil)
}

// GetNewPoolStateWithOverrides behaves like GetNewPoolState but reads all pool/plugin/tick state
// (globalState, liquidity, volatility-oracle timepoints, dynamic/sliding fee config, ticks) under the given
// state overrides, so the dynamic fee recomputed from the resulting Extra reflects the post-override
// (e.g. post-victim-tx) storage rather than the latest on-chain state.
func (t *PoolTracker) GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool,
	params sourcePool.GetNewPoolStateWithOverridesParams) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Logs, nil, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs, overrides)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, logs, blockHeaders, overrides)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// The lens read the ticks at the block it read the state, which is what refetching is for.
	if t.config.UseLens || len(p.Extra) == 0 {
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

	refetchedTicks, err := t.queryRPCTicksByIndexes(ctx, p.Address, ticksToRefetch, p.BlockNumber, nil)
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

func (t *PoolTracker) newTicksBasedPool(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	overrides map[common.Address]gethclient.OverrideAccount) (tickspkg.TicksBasedPool,
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

	blockNumber := tickspkg.ReadBlock(logs, p.BlockNumber)
	ticksBasedPool.BlockNumber = blockNumber

	ticks, err := t.fetchTicksFromLogs(ctx, p.Address, logs, blockNumber, overrides)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to FetchTicksFromLogs")
		return ticksBasedPool, err
	}

	if len(ticks) == 0 {
		return ticksBasedPool, nil
	}

	if err := tickspkg.ValidatePoolTicks(ticksBasedPool, ticks); err != nil {
		l.WithFields(logger.Fields{
			"numTicks": len(ticks),
			"error":    err,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		// Re-reading the stored indexes cannot find a tick the stored set never had; the lens
		// reads the pool's whole set. A simulation under state overrides keeps to the indexes,
		// which the overrides apply to and the lens call would not.
		if overrides == nil && t.config.UseLens && t.readAllTicks(ctx, &ticksBasedPool, l) {
			return ticksBasedPool, nil
		}

		l.WithFields(logger.Fields{
			"numTicks": len(ticksBasedPool.Ticks),
		}).Info("fetch all ticks for pool")

		ticks, err = t.fetchAllTicksForPool(ctx, ticksBasedPool, ticks, overrides)
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

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log, blockHeaders map[uint64]entity.BlockHeader,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	// Always fetch scalar state at latest, never at this batch's own block: see the
	// comment on the equivalent fetch in uniswap/v3/pool_tracker.go's updateState.
	rpcState, err := t.FetchRPCData(ctx, &p, 0, overrides)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch state from RPC")
		return p, err
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
		Timepoints:       rpcState.Timepoints,
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

	p.SwapFee = float64(rpcState.State.LastFee)
	p.Extra = string(extraBytes)
	p.Timestamp = tickspkg.EstimateLastActivityTime(&p, logs, blockHeaders)
	p.Reserves = entity.PoolReserves{
		rpcState.Reserve0.String(),
		rpcState.Reserve1.String(),
	}
	p.BlockNumber = max(p.BlockNumber, lo.LastOrEmpty(logs).BlockNumber)

	return p, nil
}

// fetchTicksFromLogs reads the ticks logs touch at blockNumber (see tickspkg.ReadBlock).
func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context, address string, logs []ethtypes.Log, blockNumber uint64,
	overrides map[common.Address]gethclient.OverrideAccount,
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

	return t.readTicks(ctx, address, tickIndexes, blockNumber, overrides)
}

// readAllTicks replaces p's ticks with the pool's whole set, read through the lens at p's block, and
// reports whether it did. The read goes through the guard, so it never makes the update wait: when
// the guard declines it, or it fails, the caller refetches the stored ticks instead.
func (t *PoolTracker) readAllTicks(ctx context.Context, p *tickspkg.TicksBasedPool, l logger.Logger) bool {
	var block *big.Int
	if p.BlockNumber > 0 {
		block = new(big.Int).SetUint64(p.BlockNumber)
	}
	ran, err := t.lensGuard.Try(ctx, p.Address, func(ctx context.Context) error {
		snap, err := t.snapshot(ctx, p.Address, block)
		if err != nil {
			return err
		}
		p.Ticks = make(map[int]tickspkg.Tick, len(snap.ticks))
		for _, tick := range snap.ticks {
			p.Ticks[tick.TickIdx] = tick
		}
		return nil
	})
	if ran && err == nil {
		return true
	}
	l.WithFields(logger.Fields{"ran": ran, "error": err}).
		Warn("ticks not read through the lens; refetching the stored ones")
	return false
}

func (t *PoolTracker) fetchAllTicksForPool(
	ctx context.Context,
	pool tickspkg.TicksBasedPool,
	ticksFromLogs []tickspkg.Tick,
	overrides map[common.Address]gethclient.OverrideAccount,
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

	ticksFromPool, err := t.readTicks(ctx, pool.Address, tickIdsFromPool, pool.BlockNumber, overrides)
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
			metrics.IncrUnprocessedEventTopic(DexType, event.Topics[0].Hex())
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
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		return t.queryRPCTicksByChunk(ctx, address, tickIndexes, blockNumber, overrides)
	}

	totalTicks := len(tickIndexes)
	ticks := make([]tickspkg.Tick, 0, totalTicks)
	for i := 0; i < totalTicks; i += tickChunkSize {
		toIdx := min(i+tickChunkSize, totalTicks)

		newTicks, err := t.queryRPCTicksByChunk(ctx, address, tickIndexes[i:toIdx], blockNumber, overrides)
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
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]tickspkg.Tick, error) {
	tickResponses := make([]TicksResp, len(ticks))
	ticksRequest := t.EthrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	ticksRequest.SetOverrides(overrides)
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
			return t.queryRPCTicksByChunk(ctx, addr, ticks, 0, overrides)
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
