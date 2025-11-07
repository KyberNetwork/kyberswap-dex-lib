package uniswapv3

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/pool"
)

var _ = pooltrack.RegisterTicksBasedFactoryCEG0(DexTypeUniswapV3, NewTracker)

type Tracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

func NewTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *Tracker {
	return &Tracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (t *Tracker) GetNewState(ctx context.Context, p entity.Pool, logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader) (entity.Pool, error) {
	if len(logs) == 0 {
		return p, nil
	}

	l := logger.WithFields(logger.Fields{
		"address":  p.Address,
		"exchange": p.Exchange,
	})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs, l)
	if err != nil {
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, logs, blockHeaders, l)
}

func (t *Tracker) FetchPoolTicks(ctx context.Context, p entity.Pool) (entity.Pool, error) {
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

	refetchedTicks, err := t.queryTicksFromRPC(ctx, p.Address, ticksToRefetch, p.BlockNumber)
	if err != nil {
		return entity.Pool{}, err
	}

	// convert back to uniswap v3 ticks
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

func (t *Tracker) newTicksBasedPool(
	ctx context.Context,
	p entity.Pool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	ticksBasedPool, err := tickspkg.NewTicksBasedPool(p)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to transform entity pool to ticks based pool")
		return tickspkg.TicksBasedPool{}, err
	}

	blockNumber := eth.GetBlockNumberFromLogs(logs)

	ticksBasedPool.BlockNumber = blockNumber

	// Log ordering: [optional empty log] + [logs from reverted blocks] + [logs from new blocks]
	// If reorg happens, only extract affected tick ids from logs and fetch their state from RPC
	if eth.HasRevertedLog(logs) {
		return t.fetchTicksFromLogs(ctx, ticksBasedPool, logs, l)
	}

	return t.computeTicksFromLogs(ctx, ticksBasedPool, logs, l)
}

func (t *Tracker) fetchTicksFromLogs(
	ctx context.Context,
	tickBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	affectedTickIds, err := t.getAffectedTickIdsFromLogs(logs)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get affected tick IDs from logs")
		return tickBasedPool, err
	}

	if len(affectedTickIds) == 0 {
		return tickBasedPool, nil
	}

	l.WithFields(logger.Fields{
		"affectedTicks": affectedTickIds,
		"blockNumber":   tickBasedPool.BlockNumber,
	}).Info("fetching affected ticks from RPC for reverted blocks")

	affectedTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, affectedTickIds, tickBasedPool.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch affected ticks from RPC")
		return tickBasedPool, err
	}

	updateTicksMap(tickBasedPool.Ticks, affectedTicks)
	if tickBasedPool.HasValidTicks() {
		return tickBasedPool, err
	}

	l.WithFields(logger.Fields{
		"affectedTicks": affectedTickIds,
		"blockNumber":   tickBasedPool.BlockNumber,
	}).Warn("invalid pool ticks data after fetching ticks from logs")

	allTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, lo.Keys(tickBasedPool.Ticks), tickBasedPool.BlockNumber)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to fetch all ticks from RPC")
		return tickBasedPool, err
	}

	updateTicksMap(tickBasedPool.Ticks, allTicks)
	if !tickBasedPool.HasAllValidTicks() {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("invalid pool ticks data after fetching all ticks stored in pool")
		return tickBasedPool, err
	}

	return tickBasedPool, nil
}

func (t *Tracker) computeTicksFromLogs(
	ctx context.Context,
	tickBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	l logger.Logger,
) (tickspkg.TicksBasedPool, error) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})

	invalidTickSet := make(map[int]struct{})
	affectedTickSet := make(map[int]struct{})

	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		lower, upper, liquidityDelta, err := t.extractEventData(event)
		if err != nil {
			l.WithFields(logger.Fields{
				"blockNumber": event.BlockNumber,
				"logIndex":    event.Index,
				"error":       err,
			}).Error("failed to extract event data")
			continue
		}

		if liquidityDelta.Sign() == 0 {
			continue
		}

		affectedTickSet[lower] = struct{}{}
		affectedTickSet[upper] = struct{}{}

		if !t.applyLiquidityChange(tickBasedPool.Ticks, lower, liquidityDelta, true) {
			invalidTickSet[lower] = struct{}{}
		}
		if !t.applyLiquidityChange(tickBasedPool.Ticks, upper, liquidityDelta, false) {
			invalidTickSet[upper] = struct{}{}
		}
	}

	if len(affectedTickSet) == 0 {
		return tickBasedPool, nil
	}

	if !tickBasedPool.HasValidTicks() || len(invalidTickSet) > 0 {
		invalidTickIds := lo.Keys(invalidTickSet)
		affectedTickIds := lo.Keys(affectedTickSet)

		logFields := logger.Fields{
			"affectedTicks": affectedTickIds,
			"blockNumber":   tickBasedPool.BlockNumber,
		}
		if len(invalidTickIds) > 0 {
			logFields["invalidTicks"] = invalidTickIds
		}
		l.WithFields(logFields).Warn("tick state accumulated from logs invalid, fetching affected ticks from RPC")

		affectedTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, affectedTickIds, tickBasedPool.BlockNumber)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to refetch affected ticks from RPC")
			return tickBasedPool, err
		}

		updateTicksMap(tickBasedPool.Ticks, affectedTicks)

		if tickBasedPool.HasValidTicks() {
			return tickBasedPool, nil
		}

		l.WithFields(logger.Fields{
			"affectedTicks": affectedTickIds,
			"blockNumber":   tickBasedPool.BlockNumber,
		}).Warn("invalid pool ticks data after fetching ticks from logs")

		allTicks, err := t.queryTicksFromRPC(ctx, tickBasedPool.Address, lo.Keys(tickBasedPool.Ticks), tickBasedPool.BlockNumber)
		if err != nil {
			l.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to refetch all ticks from RPC")
			return tickBasedPool, err
		}

		updateTicksMap(tickBasedPool.Ticks, allTicks)

		if !tickBasedPool.HasAllValidTicks() {
			l.WithFields(logger.Fields{
				"blockNumber": tickBasedPool.BlockNumber,
			}).Error("invalid pool ticks data after fetching all ticks stored in pool")

			return tickBasedPool, err
		}
	}

	return tickBasedPool, nil
}

func updateTicksMap(ticksMap map[int]tickspkg.Tick, newTicks []tickspkg.Tick) {
	for _, tick := range newTicks {
		ticksMap[tick.TickIdx] = tick
	}
}

func (t *Tracker) updateState(
	ctx context.Context,
	p entity.Pool,
	ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader,
	l logger.Logger,
) (entity.Pool, error) {
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

	extraBytes, err := json.Marshal(Extra{
		Liquidity:    rpcState.Liquidity,
		SqrtPriceX96: rpcState.Slot0.SqrtPriceX96,
		TickSpacing:  rpcState.TickSpacing.Uint64(),
		Tick:         rpcState.Slot0.Tick,
		Ticks:        entityPoolTicks,
	})
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		rpcState.Reserve0.String(),
		rpcState.Reserve1.String(),
	}
	p.Timestamp = t.estimateLastActivityTime(&p, logs, blockHeaders)

	return p, nil
}

func (t *Tracker) getAffectedTickIdsFromLogs(logs []ethtypes.Log) ([]int, error) {
	affectedTickIds := make(map[int]struct{})

	for _, event := range logs {
		if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
			continue
		}

		lower, upper, liquidityDelta, err := t.extractEventData(event)
		if err != nil {
			return nil, err
		}

		if liquidityDelta.Sign() == 0 {
			continue
		}

		affectedTickIds[lower] = struct{}{}
		affectedTickIds[upper] = struct{}{}
	}

	return lo.Keys(affectedTickIds), nil
}

func (t *Tracker) extractEventData(event ethtypes.Log) (int, int, *big.Int, error) {
	if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
		return 0, 0, big.NewInt(0), nil
	}

	switch event.Topics[0] {
	case abis.UniswapV3PoolABI.Events["Mint"].ID:
		mint, err := abis.UniswapV3PoolFilterer.ParseMint(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(mint.TickLower.Int64()), int(mint.TickUpper.Int64()), mint.Amount, nil

	case abis.UniswapV3PoolABI.Events["Burn"].ID:
		burn, err := abis.UniswapV3PoolFilterer.ParseBurn(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(burn.TickLower.Int64()), int(burn.TickUpper.Int64()), burn.Amount.Neg(burn.Amount), nil

	default:
		// metrics.IncrUnprocessedEventTopic(pooltypes.PoolTypes.UniswapV3, event.Topics[0].Hex())
		return 0, 0, big.NewInt(0), nil
	}
}

func (t *Tracker) applyLiquidityChange(
	ticks map[int]tickspkg.Tick,
	tickIdx int,
	liquidityDelta *big.Int,
	isLower bool,
) (isValid bool) {
	tick, exists := ticks[tickIdx]
	if !exists {
		tick = tickspkg.Tick{
			TickIdx:        tickIdx,
			LiquidityGross: big.NewInt(0),
			LiquidityNet:   big.NewInt(0),
		}
	}

	var newLiquidityGross big.Int
	newLiquidityGross.Add(tick.LiquidityGross, liquidityDelta)

	// exception: liquidityGross should never be negative
	if newLiquidityGross.Sign() < 0 {
		return false
	}

	tick.LiquidityGross.Set(&newLiquidityGross)

	if isLower {
		tick.LiquidityNet.Add(tick.LiquidityNet, liquidityDelta)
	} else {
		tick.LiquidityNet.Sub(tick.LiquidityNet, liquidityDelta)
	}

	ticks[tickIdx] = tick

	return true
}

// queryRPCTicksByIndexes returns ticks data of `tickIndexes` in pool `address` at `blockNumber`.
// If `blockNumber` == 0, it returns the latest ticks data.
func (t *Tracker) queryTicksFromRPC(
	ctx context.Context,
	address string,
	tickIndexes []int,
	blockNumber uint64,
) ([]tickspkg.Tick, error) {
	if len(tickIndexes) <= tickChunkSize {
		return t.queryRPCTicksByChunk(ctx, address, tickIndexes, blockNumber)
	}

	var result []tickspkg.Tick
	for i := 0; i < len(tickIndexes); i += tickChunkSize {
		end := min(i+tickChunkSize, len(tickIndexes))
		ticks, err := t.queryRPCTicksByChunk(ctx, address, tickIndexes[i:end], blockNumber)
		if err != nil {
			return nil, err
		}

		result = append(result, ticks...)
	}

	return result, nil
}

// queryRPCTicksByChunk returns univ3 Ticks data.
func (t *Tracker) queryRPCTicksByChunk(
	ctx context.Context,
	address string,
	ticks []int,
	blockNumber uint64,
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
			ABI:    abis.UniswapV3PoolABI,
			Target: address,
			Method: methodTicks,
			Params: []any{big.NewInt(int64(tick))},
		}, []any{&tickResponses[id]})
	}

	if _, err := ticksRequest.Aggregate(); err != nil {
		if blockNumber > 0 && tickspkg.IsMissingTrieNodeError(err) {
			// Re-query ticks data with latest block number
			return t.queryRPCTicksByChunk(ctx, address, ticks, 0)
		}

		return nil, fmt.Errorf("failed to process aggregate to get ticks: %w", err)
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

func (t *Tracker) estimateLastActivityTime(p *entity.Pool, logs []ethtypes.Log,
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

func (t *Tracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	l.Info("Start getting new state of pool")

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to get block number")
		return entity.Pool{}, err
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
		// Ad-hoc logic to handle edge case on Optimism
		// Link to issue: https://www.notion.so/kybernetwork/Aggregator-1-20-defect-1caec6062f9d4da0918fc3443e6e1963#0810d1462cc14f0a9465f935c9e641fe
		// TLDR: Optimism has some pre-genesis Uniswap V3 pool. Subgraph does not have data for these pools
		// So we have to fetch ticks data from the TickLens smart contract (which is slower).
		if t.config.AlwaysUseTickLens || lo.Contains[string](t.config.preGenesisPoolIDs, p.Address) {
			poolTicks, err = ticklens.GetPoolTicksFromSC(ctx, t.ethrpcClient, t.config.TickLensAddress, p, nil)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to call SC for pool ticks")
			}
		} else {
			// If pool is not pre-genesis, fetch from subgraph
			poolTicks, err = t.getPoolTicks(ctx, p.Address)
			if err != nil {
				l.WithFields(logger.Fields{
					"error": err,
				}).Error("failed to query subgraph for pool ticks")
			}
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
		Liquidity:    rpcData.Liquidity,
		TickSpacing:  rpcData.TickSpacing.Uint64(),
		SqrtPriceX96: rpcData.Slot0.SqrtPriceX96,
		Tick:         rpcData.Slot0.Tick,
		Ticks:        ticks,
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

	l.Infof("Finish updating state of pool")

	return p, nil
}

func (t *Tracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (*FetchRPCResult, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	var (
		liquidity   *big.Int
		slot0       Slot0
		tickSpacing *big.Int
		reserve0    = zeroBI
		reserve1    = zeroBI
	)

	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		rpcRequest.SetBlockNumber(&blockNumberBI)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetLiquidity,
	}, []any{&liquidity})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodGetSlot0,
	}, []any{&slot0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    abis.UniswapV3PoolABI,
		Target: p.Address,
		Method: methodTickSpacing,
	}, []any{&tickSpacing})

	if len(p.Tokens) == 2 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[0].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve0})

		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[1].Address,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&reserve1})
	}

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to process tryAggregate")
		return nil, err
	}

	return &FetchRPCResult{
		Liquidity:   liquidity,
		Slot0:       slot0,
		TickSpacing: tickSpacing,
		Reserve0:    reserve0,
		Reserve1:    reserve1,
	}, err
}

func (t *Tracker) getPoolTicks(ctx context.Context, poolAddress string) ([]TickResp, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": poolAddress,
		"dexID":       t.config.DexID,
	})

	allowSubgraphError := t.config.IsAllowSubgraphError()
	lastTickIdx := ""
	var ticks []TickResp

	for {
		req := graphqlpkg.NewRequest(getPoolTicksQuery(allowSubgraphError, poolAddress, lastTickIdx))

		var resp struct {
			Ticks []TickResp `json:"ticks"`
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
