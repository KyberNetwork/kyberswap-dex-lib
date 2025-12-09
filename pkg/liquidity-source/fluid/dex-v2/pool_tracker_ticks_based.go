package dexv2

import (
	"context"
	"math/big"
	"sort"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

func (t *PoolTracker) FetchRPCData(ctx context.Context, p *entity.Pool, blockNumber uint64) (Extra, error) {
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"dexID":       t.config.DexID,
	})

	res, err := t.fetchRPCData(ctx, *p, blockNumber, nil)
	if err != nil {
		l.WithFields(logger.Fields{
			"error":       err,
			"blockNumber": blockNumber,
		}).Error("failed to fetch pool state from RPC")
		return Extra{}, err
	}

	return res, nil
}

func (t *PoolTracker) FetchPoolTicks(ctx context.Context, p entity.Pool,
) (entity.Pool, error) {
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

func (t *PoolTracker) GetNewState(
	ctx context.Context,
	p entity.Pool,
	logs []ethtypes.Log,
	blockHeaders map[uint64]entity.BlockHeader,
) (entity.Pool, error) {
	if len(logs) == 0 {
		return p, nil
	}

	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"exchange":    p.Exchange,
	})

	ticksBasedPool, err := t.newTicksBasedPool(ctx, p, logs)
	if err != nil {
		l.Error(err.Error())
		return p, err
	}

	return t.updateState(ctx, p, ticksBasedPool, logs, blockHeaders)
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

func (t *PoolTracker) fetchTicksFromLogs(
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
			"blockNumber": tickBasedPool.BlockNumber,
		}).Error("invalid pool ticks data after fetching all ticks stored in pool")

		return tickBasedPool, err
	}

	return tickBasedPool, nil
}

func (t *PoolTracker) computeTicksFromLogs(
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

func (t *PoolTracker) getAffectedTickIdsFromLogs(logs []ethtypes.Log) ([]int, error) {
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

func (t *PoolTracker) extractEventData(event ethtypes.Log) (int, int, *big.Int, error) {
	if len(event.Topics) == 0 || eth.IsZeroAddress(event.Address) {
		return 0, 0, big.NewInt(0), nil
	}

	switch event.Topics[0] {
	case abis.DexV2ABI.Events["LogDeposit"].ID:
		deposit, err := abis.DexV2PoolFilterer.ParseLogDeposit(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(deposit.TickLower.Int64()), int(deposit.TickUpper.Int64()), deposit.LiquidityIncreaseRaw, nil

	case abis.DexV2ABI.Events["LogWithdraw"].ID:
		withdraw, err := abis.DexV2PoolFilterer.ParseLogWithdraw(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(withdraw.TickLower.Int64()), int(withdraw.TickUpper.Int64()), new(big.Int).Neg(withdraw.LiquidityDecreaseRaw), nil

	case abis.DexV2ABI.Events["LogBorrow"].ID:
		borrow, err := abis.DexV2PoolFilterer.ParseLogBorrow(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(borrow.TickLower.Int64()), int(borrow.TickUpper.Int64()), borrow.LiquidityIncreaseRaw, nil

	case abis.DexV2ABI.Events["LogPayback"].ID:
		payback, err := abis.DexV2PoolFilterer.ParseLogPayback(event)
		if err != nil {
			return 0, 0, nil, err
		}
		return int(payback.TickLower.Int64()), int(payback.TickUpper.Int64()), new(big.Int).Neg(payback.LiquidityDecreaseRaw), nil

	default:
		return 0, 0, big.NewInt(0), nil
	}
}

func (t *PoolTracker) applyLiquidityChange(
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

func (t *PoolTracker) queryTicksFromRPC(
	ctx context.Context,
	address string,
	tickIndexes []int,
	blockNumber uint64,
) ([]tickspkg.Tick, error) {
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

func (t *PoolTracker) updateState(ctx context.Context, p entity.Pool, ticksBasedPool tickspkg.TicksBasedPool,
	logs []ethtypes.Log, blockHeaders map[uint64]entity.BlockHeader) (entity.Pool, error) {
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

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	extra.Liquidity = rpcState.Liquidity
	extra.SqrtPriceX96 = rpcState.SqrtPriceX96
	extra.Tick = rpcState.Tick

	extra.Token0ExchangePricesAndConfig = rpcState.Token0ExchangePricesAndConfig
	extra.Token1ExchangePricesAndConfig = rpcState.Token1ExchangePricesAndConfig

	extra.Ticks = entityPoolTicks

	reserve0, reserve1 := extractTokenReserves(extra.Reserves)
	extra.Reserves = nil // clear reserves to avoid redundancy in extra

	extraBytes, err := json.Marshal(extra)

	if err != nil {
		l.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = t.estimateLastActivityTime(&p, logs, blockHeaders)
	p.Reserves = []string{reserve0.String(), reserve1.String()}

	return p, err
}

// queryRPCTicksByChunk returns univ3 Ticks data.
func (t *PoolTracker) queryRPCTicksByChunk(
	ctx context.Context, addr string, ticks []int, blockNumber uint64,
) ([]tickspkg.Tick, error) {
	dexID, dexType := parseFluidDexV2PoolAddress(addr)

	liquidityNet := make([]*big.Int, len(ticks))
	liquidityGross := make([]*big.Int, len(ticks))

	ticksRequest := t.ethrpcClient.NewRequest()
	ticksRequest.SetContext(ctx)
	if blockNumber > 0 {
		var blockNumberBI big.Int
		blockNumberBI.SetUint64(blockNumber)
		ticksRequest.SetBlockNumber(&blockNumberBI)
	}

	for id, tickIdx := range ticks {
		slot := calculateTripleMappingStorageSlot(
			DEX_V2_TICK_DATA_MAPPING_SLOT,
			dexType,
			common.HexToHash(dexID),
			tickIdx,
		)
		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    abis.DexV2ABI,
			Target: t.config.Dex,
			Method: "readFromStorage",
			Params: []any{slot},
		}, []any{&liquidityNet[id]})

		slot = calculateTripleMappingStorageSlot(
			DEX_V2_TICK_LIQUIDITY_GROSS_MAPPING_SLOT,
			dexType,
			common.HexToHash(dexID),
			tickIdx,
		)

		ticksRequest.AddCall(&ethrpc.Call{
			ABI:    abis.DexV2ABI,
			Target: t.config.Dex,
			Method: "readFromStorage",
			Params: []any{slot},
		}, []any{&liquidityGross[id]})
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
	for id := range len(liquidityGross) {
		curLiquidityNet := liquidityNet[id]
		if curLiquidityNet.Cmp(two255) > 0 {
			curLiquidityNet.Sub(curLiquidityNet, two256)
		}

		result[id] = tickspkg.Tick{
			TickIdx:        ticks[id],
			LiquidityGross: liquidityGross[id],
			LiquidityNet:   curLiquidityNet,
		}
	}

	return result, nil
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
