package uniswapv4

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)
var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexType, NewPoolTracker)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to unmarshal static extra")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	hookParam := &HookParam{Cfg: t.config, RpcClient: t.ethrpcClient, Pool: p}
	hook, _ := GetHook(hookAddress, hookParam)

	result := &FetchRPCResult{
		TickSpacing: staticExtra.TickSpacing,
	}
	rpcRequests := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		rpcRequests.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: t.config.StateViewAddress,
		Method: "getLiquidity",
		Params: []any{eth.StringToBytes32(p.Address)},
	}, []any{&result.Liquidity})

	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    stateViewABI,
		Target: t.config.StateViewAddress,
		Method: "getSlot0",
		Params: []any{eth.StringToBytes32(p.Address)},
	}, []any{&result.Slot0})

	res, err := rpcRequests.Aggregate()
	if err != nil {
		return result, err
	}

	if result.Reserves, err = hook.GetReserves(ctx, hookParam); err != nil {
		return nil, err
	}
	if result.Reserves == nil { // default implementation is to estimate from liquidity and sqrtPriceX96
		var reserve0, reserve1 big.Int
		if result.Slot0.SqrtPriceX96.Sign() != 0 { // reserve0 = liquidity / sqrtPriceX96 * Q96
			reserve0.Mul(result.Liquidity, Q96)
			reserve0.Div(&reserve0, result.Slot0.SqrtPriceX96)
		}
		// reserve1 = liquidity * sqrtPriceX96 / Q96
		reserve1.Mul(result.Liquidity, result.Slot0.SqrtPriceX96)
		reserve1.Div(&reserve1, Q96)
		result.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	}

	hookParam.BlockNumber = res.BlockNumber
	result.HookExtra, err = hook.Track(ctx, hookParam)
	return result, err
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})
	l.Info("Start getting new state of univ4 pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
	}

	var (
		rpcData   *FetchRPCResult
		poolTicks []ticklens.TickResp
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
		if t.config.FetchTickFromStateView {
			poolTicks, err = t.getPoolTicksFromStateView(ctx, p, param)
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

	var ticks []Tick
	for _, tickResp := range poolTicks {
		tick, err := transformTickRespToTick(tickResp)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to transform tickResp to tick")
			continue
		}

		ticks = append(ticks, tick)
	}

	extraBytes, err := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			Liquidity:    rpcData.Liquidity,
			TickSpacing:  uint64(rpcData.TickSpacing),
			SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
			Tick:         rpcData.Slot0.Tick,
			Ticks:        ticks,
		},
		HookExtra: rpcData.HookExtra,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = rpcData.Reserves
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	l.Infof("Finish updating state of pool")
	return p, nil
}

// getPoolTicks
func (t *PoolTracker) getPoolTicks(ctx context.Context, poolAddress string) ([]ticklens.TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []ticklens.TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []ticklens.TickResp `json:"ticks"`
		}

		if err := t.graphqlClient.Run(ctx, req, &resp); err != nil {
			// Workaround at the moment to live with the error subgraph on Arbitrum
			if allowSubgraphError {
				if resp.Ticks == nil {
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

		if len(resp.Ticks) == 0 {
			break
		}

		ticks = append(ticks, resp.Ticks...)

		if len(resp.Ticks) < graphFirstLimit {
			break
		}

		lastTickIdx = resp.Ticks[len(resp.Ticks)-1].TickIdx
	}

	return ticks, nil
}

type stateViewTick struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
}

func (t *PoolTracker) getPoolTicksFromStateView(
	ctx context.Context,
	p entity.Pool,
	param poolpkg.GetNewPoolStateParams,
) ([]ticklens.TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, errors.New("failed to unmarshal pool extra")
	}

	changedTicks := ticklens.GetChangedTicks(param.Logs)
	l.Infof("Fetch changed ticks %v", changedTicks)

	changedTicksCount := len(changedTicks)
	if changedTicksCount == 0 || changedTicksCount > maxChangedTicks {
		return nil, ErrTooManyChangedTicks
	}

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))

	stateViewTicks := make([]stateViewTick, changedTicksCount)
	for i, tickIdx := range changedTicks {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    stateViewABI,
			Target: t.config.StateViewAddress,
			Method: "getTickInfo",
			Params: []any{eth.StringToBytes32(p.Address), big.NewInt(tickIdx)},
		}, []any{&stateViewTicks[i]})
	}

	resp, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	resTicks := make(map[int64]stateViewTick, len(resp.Request.Calls))
	for i, tick := range stateViewTicks {
		resTicks[changedTicks[i]] = tick
	}

	combined := make([]ticklens.TickResp, 0, len(changedTicks)+len(extra.Ticks))
	for _, t := range extra.Ticks {
		tIdx := int64(t.Index)
		if slices.Contains(changedTicks, tIdx) {
			tick := resTicks[tIdx]
			if tick.LiquidityNet == nil || tick.LiquidityNet.Sign() == 0 {
				// some changed ticks might be consumed entirely, delete them
				logger.Debugf("deleted tick %v %v", p.Address, t)
				continue
			}

			// changed, use new value
			combined = append(combined, ticklens.TickResp{
				TickIdx:        strconv.FormatInt(tIdx, 10),
				LiquidityGross: tick.LiquidityGross.String(),
				LiquidityNet:   tick.LiquidityNet.String(),
			})
		} else {
			// use old value
			combined = append(combined, ticklens.TickResp{
				TickIdx:        strconv.Itoa(t.Index),
				LiquidityGross: t.LiquidityGross.String(),
				LiquidityNet:   t.LiquidityNet.String(),
			})
		}
	}

	// Sort the ticks because function NewTickListDataProvider needs
	sort.SliceStable(combined, func(i, j int) bool {
		iTick, _ := strconv.Atoi(combined[i].TickIdx)
		jTick, _ := strconv.Atoi(combined[j].TickIdx)

		return iTick < jTick
	})

	return combined, nil
}

func transformTickRespToTick(tickResp ticklens.TickResp) (Tick, error) {
	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}

func (t *PoolTracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader) (entity.Pool, error) {
	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address":  p.Address,
			"exchange": p.Exchange,
		}).Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, logs, blockHeaders)
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	// Extract current ticks from entity pool extra
	var extra Extra
	if len(p.Extra) > 0 {
		err := json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return entity.Pool{}, err
		}
	}

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
		return entity.Pool{}, err
	}

	// convert back to uniswap v4 ticks
	entityPoolTicks := make([]Tick, 0, len(refetchedTicks))
	for _, tick := range refetchedTicks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, Tick{
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
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) newTicksBasedPool(
	ctx context.Context,
	p entity.Pool,
	logs []ethtypes.Log,
) (tickspkg.TicksBasedPool, error) {
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

func (t *PoolTracker) fetchTicksFromLogs(
	ctx context.Context,
	address string,
	logs []ethtypes.Log,
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

// getTickIndexesFromLogs returns all tick indexes from logs.
func (t *PoolTracker) getTickIndexesFromLogs(logs []ethtypes.Log) ([]int, error) {
	tickSet := make(map[int]struct{})
	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		switch event.Topics[0] {
		case poolManagerABI.Events["ModifyLiquidity"].ID:
			mint, err := poolManagerFilterer.ParseModifyLiquidity(event)
			if err != nil {
				logger.WithFields(logger.Fields{
					"event": event,
					"error": err,
				}).Error("failed to parse ModifyLiquidity event")
				return nil, err
			}

			logger.WithFields(logger.Fields{
				"address": event.Address,
				"event":   mint,
			}).Debug("decode ModifyLiquidity event")

			tickSet[int(mint.TickLower.Int64())] = struct{}{}
			tickSet[int(mint.TickUpper.Int64())] = struct{}{}

		default:
			// metrics.IncrUnprocessedEventTopic(pooltypes.PoolTypes.UniswapV4, event.Topics[0].Hex())
		}
	}

	ticks := make([]int, 0, len(tickSet))
	for tick := range tickSet {
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

// queryRPCTicksByIndexes returns ticks data of `tickIndexes` in pool `address` at `blockNumber`.
// If `blockNumber` == 0, it returns the latest ticks data.
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

// queryRPCTicksByChunk returns univ4 Ticks data.
func (t *PoolTracker) queryRPCTicksByChunk(
	ctx context.Context, addr string, ticks []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	tickResponses := make([]TicksResp, len(ticks))
	ticksRequest := t.ethrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		ticksRequest.SetBlockNumber(&blockNumberBI)
	}

	for id, tick := range ticks {
		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    stateViewABI,
			Target: t.config.StateViewAddress,
			Method: "getTickLiquidity",
			Params: []any{stringToBytes32(addr), big.NewInt(int64(tick))},
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
			LiquidityGross: tickResponse.LiquidityGross,
			LiquidityNet:   tickResponse.LiquidityNet,
		}
	}

	return result, nil
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

func (t *PoolTracker) updateState(
	ctx context.Context,
	p entity.Pool,
	ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader,
) (entity.Pool, error) {
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

	entityPoolTicks := make([]Tick, 0, len(ticksBasedPool.Ticks))
	for _, tick := range ticksBasedPool.Ticks {
		// skip uninitialized ticks
		if tick.LiquidityGross.Sign() == 0 {
			continue
		}

		entityPoolTicks = append(entityPoolTicks, Tick{
			Index:          tick.TickIdx,
			LiquidityGross: tick.LiquidityGross,
			LiquidityNet:   tick.LiquidityNet,
		})
	}

	// Sort the ticks by tick index
	sort.Slice(entityPoolTicks, func(i, j int) bool {
		return entityPoolTicks[i].Index < entityPoolTicks[j].Index
	})

	if rpcState.Slot0.SqrtPriceX96.Sign() == 0 {
		l.Error("sqrtPriceX96 is 0")
		return p, errors.New("sqrtPriceX96 is 0")
	}

	extraBytes, err := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			Liquidity:    rpcState.Liquidity,
			SqrtPriceX96: rpcState.Slot0.SqrtPriceX96,
			TickSpacing:  uint64(rpcState.TickSpacing),
			Tick:         rpcState.Slot0.Tick,
			Ticks:        entityPoolTicks,
		},
		HookExtra: rpcState.HookExtra,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.SwapFee, _ = rpcState.Slot0.LpFee.Float64()
	p.Extra = string(extraBytes)
	p.Reserves = rpcState.Reserves
	p.Exchange = t.getExchange(&p)
	p.Timestamp = t.estimateLastActivityTime(&p, logs, blockHeaders)

	return p, nil
}

func (t *PoolTracker) getExchange(p *entity.Pool) string {
	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.Errorf("failed to unmarshal static extra data")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	hookParam := &HookParam{Cfg: t.config, RpcClient: t.ethrpcClient, Pool: p}
	hook, _ := GetHook(hookAddress, hookParam)

	return hook.GetExchange()
}

func (t *PoolTracker) estimateLastActivityTime(p *entity.Pool, logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader) int64 {
	if len(logs) > 0 && blockHeaders != nil {
		latestLog := logs[len(logs)-1]
		if blockHeader, ok := blockHeaders[latestLog.BlockNumber]; ok {
			return max(p.Timestamp, int64(blockHeader.Timestamp))
		}
	}

	// Do not update the timestamp as the pool triggered state update via a custom empty log.
	return p.Timestamp
}

func stringToBytes32(str string) [32]byte {
	var result [32]byte
	// If the string is hex encoded (starts with 0x), decode it first
	if strings.HasPrefix(str, "0x") {
		decoded, _ := hex.DecodeString(str[2:])
		copy(result[:], decoded)
	} else {
		copy(result[:], str)
	}
	return result
}
