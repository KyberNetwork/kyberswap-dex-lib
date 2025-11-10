package algebrav1

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/metrics"
	"github.com/KyberNetwork/logger"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pooltrack.RegisterFactoryCEG0(DexTypeAlgebraV1, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexTypeAlgebraV1, NewPoolTracker)

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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pool")

	var (
		rpcData   *FetchRPCResult
		poolTicks []TickResp
	)

	blockNumber, err := t.EthrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

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
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcData.Reserve0.String(),
		rpcData.Reserve1.String(),
	}
	p.BlockNumber = blockNumber

	l.WithFields(logger.Fields{
		"feeZto": rpcData.State.FeeZto,
		"feeOtz": rpcData.State.FeeOtz,
	}).Info("Finish updating state of pool")

	return p, nil
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
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
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

	_, err := rpcRequest.Aggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return res, err
	}

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
		err = t.approximateFee(ctx, p.Address, dataStorageOperator.Hex(), &res.State, res.Liquidity)
		if err != nil {
			return res, err
		}
	}

	if !res.State.Unlocked {
		l.Info("pool has been locked and not usable")
	}

	return res, err
}

func (t *PoolTracker) approximateFee(ctx context.Context, poolAddress, dataStorageOperator string,
	state *GlobalState, currentLiquidity *big.Int) error {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	// fee approximation: assume that the swap will be soon after this
	blockTimestamp := uint32(time.Now().Unix())
	yesterday := blockTimestamp - WINDOW
	timepoints, err := t.getPoolTimepoints(ctx, state.TimepointIndex, poolAddress, yesterday)
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

	volumePerLiquidityInBlock, err := t.getPoolVolumePerLiquidityInBlock(ctx, common.HexToAddress(poolAddress))
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
	yesterday uint32) (map[uint16]Timepoint, error) {
	return t.GetTimepoints(ctx, &ethrpc.Call{
		ABI:    algebraV1PoolABI,
		Target: poolAddress,
		Method: methodGetTimepoints,
	}, nil, yesterday, currentIndex, nil)
}

func (t *PoolTracker) getPoolVolumePerLiquidityInBlock(ctx context.Context, poolAddress common.Address) (*big.Int,
	error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	abiUint256, _ := abi.NewType("uint256", "", nil)
	abi := abi.Arguments{
		// 2 variables are stored in 1 slot, need to read the whole and shift out later
		{Name: "liquidity_volumePerLiquidityInBlock", Type: abiUint256},
	}

	resp, err := t.EthrpcClient.NewRequest().SetContext(ctx).GetStorageAt(
		poolAddress,
		slot3,
		abi,
	)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch pool volumePerLiquidityInBlock")
		return nil, err
	}

	if len(resp) == 1 {
		if bi, ok := resp[0].(*big.Int); ok {
			return new(big.Int).Rsh(bi, 128), nil
		}
	}
	l.WithFields(logger.Fields{
		"resp": resp,
	}).Error("failed to unmarshal volumePerLiquidityInBlock")
	return nil, ErrUnmarshalVolLiq
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
			metrics.IncrUnprocessedEventTopic(pooltypes.PoolTypes.AlgebraV1, event.Topics[0].Hex())
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
