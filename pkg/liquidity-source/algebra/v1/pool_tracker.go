package algebrav1

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexTypeAlgebraV1, NewPoolTracker)

type PoolTracker struct {
	algebra.PoolTracker[Timepoint, TimepointRPC]
	config        *Config
	graphqlClient *graphqlpkg.Client
	lensGuard     *ephemeral.Guard
	// tickTable is how the exchange's pools index their tick table, once a lens read learnt it.
	tickTable atomic.Int32
	readLens  readLens
	// readTicks reads ticks by index at a block, queryRPCTicksByIndexes but for tests.
	readTicks func(ctx context.Context, address string, tickIndexes []int, blockNumber uint64) ([]tickspkg.Tick,
		error)
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
	t.readLens = t.readLensPages
	t.readTicks = t.queryRPCTicksByIndexes
	return t
}

func (t *PoolTracker) BootstrapPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pool")

	if t.config.UseLens {
		p, _, err := t.bootstrapFromLens(ctx, p, param, nil, l)
		return p, err
	}

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	g := pool.New().WithContext(ctx)
	g.Go(func(context.Context) error {
		var err error
		rpcData, err = t.FetchRPCData(ctx, &p, 0)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to fetch data from RPC")

		}

		return err
	})
	g.Go(func(context.Context) error {
		var err error
		if t.config.AlwaysUseTickLens {
			poolTicks, err = t.getPoolTicksFromSC(ctx, p, param)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
			return err
		}

		poolTicks, err = t.getPoolTicks(ctx, p.Address)
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
		tick, err := transformTickRespToTickBigInt(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			continue
		}

		// LiquidityGross = 0 means that the tick is uninitialized
		if tick.LiquidityGross.Cmp(bignumber.ZeroBI) == 0 {
			continue
		}

		ticks = append(ticks, tick)
	}

	return withState(p, rpcData, ticks, param, 0, l)
}

var _ sourcePool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)

// NextUpdateAt asks for a pool still without state, its lens read declined by the guard, to be
// updated again as soon as the guard would take it: it may see no log for a long time, and only an
// update gives it its state. Every other pool needs no update without an event.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 {
	if t.config.UseLens && !tickspkg.HasState(p) {
		return t.lensGuard.ReadyAt(p.Address, time.Now()).Unix()
	}
	return 0
}

var _ sourcePool.IPoolTrackerWithReset = (*PoolTracker)(nil)

// ResetPoolState drops the pool's state for its next update to read it whole through the lens, at
// a block no lower than block. Without the lens nothing can read it whole, and the pool is kept.
func (t *PoolTracker) ResetPoolState(p entity.Pool, block uint64) (entity.Pool, bool) {
	if !t.config.UseLens {
		return p, false
	}
	return tickspkg.WithoutState(p, block), true
}

// bootstrapFromLens reads the pool's state and ticks through the lens at block (nil for the
// latest), then the rest of its state, which the lens does not return, at the same block. It
// returns the pool with the block the lens read it at.
func (t *PoolTracker) bootstrapFromLens(ctx context.Context, p entity.Pool, param sourcePool.GetNewPoolStateParams,
	block *big.Int, l logger.Logger) (entity.Pool, uint64, error) {
	snap, err := t.snapshot(ctx, p.Address, block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Warn("failed to read pool through the lens")
		return entity.Pool{}, 0, err
	}

	rpcData, err := t.FetchRPCData(ctx, &p, snap.block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch data from RPC")
		return entity.Pool{}, 0, err
	}
	if rpcData.Liquidity.Cmp(snap.liquidity) != 0 || rpcData.State.Tick.Int64() != int64(snap.tick) {
		return entity.Pool{}, 0, fmt.Errorf("pool %s at block %d: RPC and lens disagree on liquidity or tick",
			p.Address, snap.block)
	}

	p, err = withState(p, rpcData, entityTicks(snap.ticks), param, snap.block, l)
	return p, snap.block, err
}

// withState writes the fetched state into p. lensBlock is the block the lens read the state and
// ticks whole at, 0 when they came another way: on Arbitrum it is the chain's own block, where the
// multicall's is Ethereum's.
func withState(p entity.Pool, rpcData *FetchRPCResult, ticks []v3Entities.Tick,
	param sourcePool.GetNewPoolStateParams, lensBlock uint64, l logger.Logger) (entity.Pool, error) {
	extraBytes, err := json.Marshal(Extra{
		Liquidity:   rpcData.Liquidity,
		GlobalState: rpcData.State,
		Ticks:       ticks,
		TickSpacing: int24(rpcData.TickSpacing.Int64()),
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = max(p.Timestamp, int64(lo.LastOrEmpty(param.Logs).BlockTimestamp))
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = max(p.BlockNumber, lensBlock, lo.LastOrEmpty(param.Logs).BlockNumber, rpcData.BlockNumber)

	l.WithFields(logger.Fields{
		"feeZto": rpcData.State.FeeZto,
		"feeOtz": rpcData.State.FeeOtz,
	}).Info("Finish updating state of pool")

	return p, nil
}

// entityTicks is ticks as the pool's Extra keeps them, ascending, without the uninitialized ones.
func entityTicks(ticks []tickspkg.Tick) []v3Entities.Tick {
	out := make([]v3Entities.Tick, 0, len(ticks))
	for _, tick := range ticks {
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}
		out = append(out, v3Entities.Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var dataStorageOperator common.Address
	res := &FetchRPCResult{}

	rpcRequest := t.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	var block *big.Int
	if blockNumber > 0 {
		block = new(big.Int).SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(block)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
	}, []any{&res.Liquidity})

	// the globalstate abi are slightly different across versions
	var rpcState any
	if t.config.UseDirectionalFee {
		rpcState = &rpcGlobalStateDirFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1DirFeePoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
		}, []any{rpcState})
	} else {
		rpcState = &rpcGlobalStateSingleFee{}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    algebraV1PoolABI,
			Target: p.Address,
			Method: methodGetGlobalState,
		}, []any{rpcState})
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetDataStorageOperator,
	}, []any{&dataStorageOperator})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: p.Address,
		Method: methodGetTickSpacing,
	}, []any{&res.TickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abipkg.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abipkg.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abipkg.Erc20ABI,
			Target: p.Tokens[1].Address,
			Method: abipkg.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&res.Reserve1})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return res, err
	}
	res.BlockNumber = resp.BlockNumber.Uint64()

	if t.config.UseDirectionalFee {
		rpcStateRes := rpcState.(*rpcGlobalStateDirFee)
		res.State = GlobalState{
			Price:              rpcStateRes.Price,
			Tick:               rpcStateRes.Tick,
			FeeZto:             rpcStateRes.FeeZto,
			FeeOtz:             rpcStateRes.FeeOtz,
			TimepointIndex:     rpcStateRes.TimepointIndex,
			CommunityFeeToken0: rpcStateRes.CommunityFeeToken0,
			CommunityFeeToken1: rpcStateRes.CommunityFeeToken1,
			Unlocked:           rpcStateRes.Unlocked,
		}
	} else {
		// for v1 without directional fee, we'll use Fee for both FeeZto/FeeOtz
		rpcStateRes := rpcState.(*rpcGlobalStateSingleFee)
		res.State = GlobalState{
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

	if !t.config.SkipFeeCalculating {
		// at the block asked for, which pins the state read above; res.BlockNumber is the multicall contract's view of
		// block.number, the L1 block through Multicall3 on Arbitrum
		err = t.approximateFee(ctx, p.Address, dataStorageOperator.Hex(), &res.State, res.Liquidity, block)
		if err != nil {
			return res, err
		}
	}

	if !res.State.Unlocked {
		l.Info("pool has been locked and not usable")
	}

	return res, err
}

// approximateFee sets the fee a swap pays at block (nil for the latest block), which the pool computes again at the
// first swap of a block from the timepoints it writes then. The fee in globalState is the one the last swap paid.
func (t *PoolTracker) approximateFee(ctx context.Context, poolAddress, dataStorageOperator string,
	state *GlobalState, currentLiquidity, block *big.Int) error {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	timestamp, err := t.EthrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block).GetCurrentBlockTimestamp()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch block timestamp")
		return err
	}
	blockTimestamp := uint32(timestamp)
	yesterday := blockTimestamp - WINDOW
	timepoints, err := t.getPoolTimepoints(ctx, state.TimepointIndex, poolAddress, yesterday, block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
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
	if t.config.UseDirectionalFee {
		err = t.getPoolDirectionalFeeConfig(ctx, dataStorageOperator, &feeConfZto, &feeConfOtz)
	} else {
		err = t.getPoolFeeConfig(ctx, dataStorageOperator, &feeConf)
	}
	if err != nil {
		return err
	}

	volumePerLiquidityInBlock, err := t.getPoolVolumePerLiquidityInBlock(ctx, common.HexToAddress(poolAddress), block)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool volumePerLiquidityInBlock")
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

	if t.config.UseDirectionalFee {
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

func (t *PoolTracker) getPoolFeeConfig(ctx context.Context, dataStorageOperatorAddress string,
	feeConf *FeeConfiguration) error {
	rpcRequest := t.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DataStorageOperatorABI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfig,
	}, []any{feeConf})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dataStorageAddress": dataStorageOperatorAddress,
			"error":              err,
		}).Error("failed to fetch from data storage operator")
		return err
	}
	return nil
}

func (t *PoolTracker) getPoolDirectionalFeeConfig(ctx context.Context, dataStorageOperatorAddress string,
	feeConfZto *FeeConfiguration, feeConfOtz *FeeConfiguration) error {
	rpcRequest := t.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorABI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigZto,
	}, []any{feeConfZto},
	).AddCall(&ethrpc.Call{
		ABI:    algebraV1DirFeeDataStorageOperatorABI,
		Target: dataStorageOperatorAddress,
		Method: methodGetFeeConfigOtz,
	}, []any{feeConfOtz})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dataStorageAddress": dataStorageOperatorAddress,
			"error":              err,
		}).Error("failed to fetch from data storage operator")
		return err
	}
	return nil
}

func (t *PoolTracker) getPoolTimepoints(ctx context.Context, currentIndex uint16, poolAddress string,
	yesterday uint32, block *big.Int) (map[uint16]Timepoint, error) {
	return t.GetTimepoints(ctx, &ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: poolAddress,
		Method: methodGetTimepoints,
	}, block, yesterday, currentIndex, 0, nil, nil)
}

func (t *PoolTracker) getPoolVolumePerLiquidityInBlock(ctx context.Context, poolAddress common.Address,
	block *big.Int) (*big.Int, error) {
	// liquidity and volumePerLiquidityInBlock share slot 3, the volume in its upper 128 bits; read at the
	// state's block (ethrpc's GetStorageAt reads the latest)
	slot, err := t.EthrpcClient.GetETHClient().StorageAt(ctx, poolAddress, slot3, block)
	if err != nil {
		return nil, err
	}
	if len(slot) != common.HashLength {
		return nil, ErrUnmarshalVolLiq
	}
	return new(big.Int).SetBytes(slot[:16]), nil
}

func (t *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.AllowSubgraphError
	skip := 0
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, skip))

		var resp struct {
			Pool *SubgraphPoolTicks        `json:"pool"`
			Meta *valueobject.SubgraphMeta `json:"_meta"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
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
		resp.Meta.CheckIsLagging(t.config.DexID, poolAddress)

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
	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	if t.config.UseLens && !tickspkg.HasState(p) {
		return tickspkg.StateFromLens(ctx, t.lensGuard, p, param,
			func(ctx context.Context, block *big.Int) (entity.Pool, uint64, error) {
				return t.bootstrapFromLens(ctx, p, param, block, l)
			},
			l), nil
	}

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, param.Logs)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, param.Logs, param.BlockHeaders)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// The lens read the ticks at the block it read the state, which is what refetching is for.
	if t.config.UseLens {
		return p, nil
	}

	// Extract current ticks from entity pool extra
	var extra Extra
	if len(p.Extra) > 0 {
		err := json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return p, err
		}
	}

	// Use a map here to filter duplicated tick indexes
	ticks := map[int]struct{}{}
	for _, tick := range extra.Ticks {
		ticks[tick.Index] = struct{}{}
	}

	ticksToRefetch := make([]int, 0, len(ticks))
	for tickIdx := range ticks {
		ticksToRefetch = append(ticksToRefetch, tickIdx)
	}

	if len(ticksToRefetch) == 0 {
		return p, nil
	}

	refetchedTicks, err := t.queryRPCTicksByIndexes(ctx, p.Address, ticksToRefetch, p.BlockNumber)
	if err != nil {
		return p, err
	}

	// convert back to algebra v1 ticks (uniswapv3entities.Tick actually)
	// https://github.com/KyberNetwork/kyberswap-dex-lib/blob/599887838de51928f8d41b7c9f88434f31f1b3d8/pkg/liquidity-source/algebra/v1/pool_tracker.go#L100
	entityPoolTicks := make([]v3Entities.Tick, 0, len(refetchedTicks))
	for _, tick := range refetchedTicks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, v3Entities.Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

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

func (t *PoolTracker) newTicksBasedPool(ctx context.Context, p entity.Pool, logs []ethtypes.Log) (tickspkg.TicksBasedPool, error) {
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

	ticks, err := t.fetchTicksFromLogs(ctx, p.Address, logs, blockNumber)
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

		if t.ticksFromLens(ctx, &ticksBasedPool) {
			return ticksBasedPool, nil
		}

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

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log, blockHeaders map[uint64]entity.BlockHeader) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	})

	// Always fetch scalar state at latest, never at this batch's own block: see the
	// comment on the equivalent fetch in uniswap/v3/pool_tracker.go's updateState.
	rpcState, err := t.FetchRPCData(ctx, &p, 0)
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
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

	extraBytes, err := json.Marshal(Extra{
		Liquidity:   rpcState.Liquidity,
		GlobalState: rpcState.State,
		Ticks:       entityPoolTicks,
		TickSpacing: int32(rpcState.TickSpacing.Int64()),
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

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
) ([]tickspkg.Tick, error) {
	l := logger.WithFields(logger.Fields{
		"address": address,
	})

	if len(logs) == 0 {
		return nil, nil
	}

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

	return t.readTicks(ctx, address, tickIndexes, blockNumber)
}

// ticksFromLens replaces the pool's ticks with the lens's set once those accumulated from logs no
// longer sum to zero, and reports whether it did. Refetching the stored ticks cannot find one the
// stored set never had; the lens reads every initialized tick. It reads at the block the logs'
// ticks were read at (see tickspkg.ReadBlock), the latest when they named none.
//
// The read goes through the guard, so it never makes the update wait: when the guard declines it,
// or it fails, the stored ticks are refetched as without the lens.
func (t *PoolTracker) ticksFromLens(ctx context.Context, p *tickspkg.TicksBasedPool) bool {
	if !t.config.UseLens {
		return false
	}
	ran, err := t.lensGuard.Try(ctx, p.Address, func(ctx context.Context) error {
		var block *big.Int
		if p.BlockNumber != 0 {
			block = new(big.Int).SetUint64(p.BlockNumber)
		}
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
	logger.WithFields(logger.Fields{"address": p.Address, "ran": ran, "error": err}).
		Warn("ticks not read through the lens; refetching the stored ones")
	return false
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

	ticksFromPool, err := t.readTicks(ctx, pool.Address, tickIdsFromPool, pool.BlockNumber)
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
	tickSet := make(map[int]struct{})
	for _, event := range logs {
		if len(event.Topics) == 0 {
			continue
		}

		switch event.Topics[0] {
		case algebraV1PoolABI.Events["Mint"].ID:
			mint, err := poolFilterer.ParseMint(event)
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

		case algebraV1PoolABI.Events["Burn"].ID:
			burn, err := poolFilterer.ParseBurn(event)
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
			metrics.IncrUnprocessedEventTopic(DexTypeAlgebraV1, event.Topics[0].Hex())
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
		toIdx := min(i+tickChunkSize, totalTicks)

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
			ABI:    algebraV1PoolABI,
			Target: addr,
			Method: methodGetTicks,
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
