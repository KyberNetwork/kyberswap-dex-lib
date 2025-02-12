package integral

import (
	"context"
	"errors"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	algebra.PoolTracker[Timepoint, TimepointRPC]
	config        *Config
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		PoolTracker:   algebra.PoolTracker[Timepoint, TimepointRPC]{EthrpcClient: ethrpcClient},
		config:        cfg,
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

	req := d.EthrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}

	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolLiquidityMethod,
	}, []any{&res.Liquidity})

	rpcState := &GlobalStateFromRPC{}
	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolGlobalStateMethod,
	}, []any{rpcState})

	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
		Target: p.Address,
		Method: poolTickSpacingMethod,
	}, []any{&res.TickSpacing})

	if len(p.Tokens) == 2 {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[0].Address,
			Method: erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve0})

		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: p.Tokens[1].Address,
			Method: erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve1})
	}

	var plugin common.Address
	req.AddCall(&ethrpc.Call{
		ABI:    algebraIntegralPoolABI,
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
	res.SlidingFee = slidingFeeData
	res.DynamicFee = dynamicFeeData
	res.BlockNumber = result.BlockNumber

	return res, nil
}

func (d *PoolTracker) getPluginData(ctx context.Context, p entity.Pool, plugin string,
	blockNumber *big.Int) (map[uint16]Timepoint, VolatilityOraclePlugin, DynamicFeeConfig,
	SlidingFeeConfig, error) {
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
	} else {
		slidingFeeData, sliPost = &SlidingFeeConfig{}, func(*ethrpc.Response) error { return nil }
	}

	resp, err := req.TryAggregate()
	if err == nil {
		err = volPost(resp)
	}
	if err == nil {
		err = dynPost(resp)
	}
	if err == nil {
		err = sliPost(resp)
	}
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch plugin data")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	var extra ExtraTimepoint
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	timepoints, err := d.getTimepoints(ctx, plugin, blockNumber, volatilityOracleData.TimepointIndex, extra.Timepoints)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch timepoints data from plugin")
		return nil, VolatilityOraclePlugin{}, DynamicFeeConfig{}, SlidingFeeConfig{}, err
	}

	return timepoints, *volatilityOracleData, *dynamicFeeData, *slidingFeeData, nil
}

func (d *PoolTracker) getVolatilityOracleData(req *ethrpc.Request, pluginAddress string) (*VolatilityOraclePlugin,
	func(resp *ethrpc.Response) error) {
	var result VolatilityOraclePlugin
	callsFrom := len(req.Calls)

	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginIsInitializedMethod,
	}, []any{&result.IsInitialized})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: votalityOraclePluginLastTimepointTimestampMethod,
	}, []any{&result.LastTimepointTimestamp})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
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
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeConfigMethod,
	}, []any{&result})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeZeroToOneMethod,
	}, []any{&result.ZeroToOne})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: dynamicFeeManagerPluginFeeOneToZeroMethod,
	}, []any{&result.OneToZero})

	return &result, func(resp *ethrpc.Response) error {
		if !(resp.Result[callsFrom] || resp.Result[callsFrom+1] && resp.Result[callsFrom+2]) {
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
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginFeeFactorsMethod,
	}, []any{&result})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginPriceChangeFactorMethod,
	}, []any{&cfg.PriceChangeFactor})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
		Target: pluginAddress,
		Method: slidingFeePluginBaseFeeMethod,
	}, []any{&cfg.BaseFee})
	req.AddCall(&ethrpc.Call{
		ABI:    algebraBasePluginV2ABI,
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
		ABI:    algebraBasePluginV2ABI,
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
