package dexv2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
)

const (
	// A full bitmap scan is forced at least this often (~1h on Ethereum) as a safety net.
	ticksFullRescanBlocks = 300
	// Logs are re-checked from this many blocks before the last scan to tolerate shallow reorgs.
	ticksReorgOverlap = 8
)

// Tick liquidity (liquidityGross/liquidityNet) and the initialized-tick bitmap are written only by the
// D3 deposit/withdraw and D4 borrow/payback paths. Swaps touch a tick's fee-growth fields only, and the
// admin/controller modules never write tick data. Each of those four paths emits one of these events
// carrying the pool's dexId as an indexed topic, so their absence proves the tick list is unchanged.
func tickChangeEventIDs() []common.Hash {
	return []common.Hash{
		abis.DexV2ABI.Events["LogDeposit"].ID,
		abis.DexV2ABI.Events["LogWithdraw"].ID,
		abis.DexV2ABI.Events["LogBorrow"].ID,
		abis.DexV2ABI.Events["LogPayback"].ID,
	}
}

// canReuseTicks reports whether cached ticks scanned at scannedBlock are still valid at snapshotBlock,
// ignoring logs: it only checks that the cache exists and is fresh enough.
func canReuseTicks(prev *Extra, snapshotBlock uint64) bool {
	return prev != nil && len(prev.Ticks) > 0 && prev.TicksBlock > 0 &&
		snapshotBlock >= prev.TicksBlock && snapshotBlock-prev.TicksBlock <= ticksFullRescanBlocks
}

func ticksLogQuery(dex common.Address, dexType uint32, dexID common.Hash, scannedBlock, snapshotBlock uint64) ethereum.FilterQuery {
	from := uint64(0)
	if scannedBlock > ticksReorgOverlap {
		from = scannedBlock - ticksReorgOverlap
	}
	return ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(snapshotBlock),
		Addresses: []common.Address{dex},
		Topics: [][]common.Hash{
			tickChangeEventIDs(),
			{common.BigToHash(new(big.Int).SetUint64(uint64(dexType)))},
			{dexID},
		},
	}
}

// liquidityChanged reports whether a tick-change event moved any liquidity. The last data word is
// liquidityIncreaseRaw/liquidityDecreaseRaw; a zero there is a fee-only call (how fees are collected),
// which leaves tick gross, net and the bitmap untouched. Unexpected layouts count as changed.
func liquidityChanged(l ethtypes.Log) bool {
	const words = 8 // tickLower, tickUpper, salt, amount0, amount1, fee0, fee1, liquidity
	if len(l.Data) != 32*words {
		return true
	}
	return new(big.Int).SetBytes(l.Data[32*(words-1):]).Sign() != 0
}

// cachedTicks returns the ticks stored in the previous pool state when they are provably unchanged
// up to snapshotBlock. Any doubt (no cache, stale cache, RPC error, any matching log) returns false
// so the caller falls back to a full bitmap scan.
func (t *PoolTracker) cachedTicks(ctx context.Context, p entity.Pool, snapshotBlock uint64) ([]Tick, uint64, bool) {
	var prev Extra
	if p.Extra == "" || json.Unmarshal([]byte(p.Extra), &prev) != nil || !canReuseTicks(&prev, snapshotBlock) {
		return nil, 0, false
	}
	dexID, dexType := parseFluidDexV2PoolAddress(p.Address)
	logs, err := t.ethrpcClient.GetETHClient().FilterLogs(ctx,
		ticksLogQuery(common.HexToAddress(t.config.Dex), dexType, common.HexToHash(dexID), prev.TicksBlock, snapshotBlock))
	if err != nil {
		logger.Warnf("[%s] tick log check failed for %s, rescanning bitmap: %v", t.config.DexID, p.Address, err)
		return nil, 0, false
	}
	for _, l := range logs {
		if liquidityChanged(l) {
			return nil, 0, false
		}
	}
	return prev.Ticks, prev.TicksBlock, true
}
