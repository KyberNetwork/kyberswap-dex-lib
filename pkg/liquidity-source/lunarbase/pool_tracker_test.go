package lunarbase

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestGetNewPoolStatePrefersLogsOverFlashCache(t *testing.T) {
	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:      uint256.NewInt(1),
		FeeAskX24:         0,
		FeeBidX24:         1,
		LatestUpdateBlock: 10,
		BlockDelay:        5,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	poolEntity := entity.Pool{
		Address:     "0x00003bf45ce34bf1bea78669f9a40ee630e11b99",
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 10,
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       string(extraBytes),
	}

	subscriberInstance = &FlashBlockSubscriber{
		latestState: &poolState{
			SqrtPriceX96:      uint256.NewInt(2),
			FeeAskX24:         0,
			FeeBidX24:         2,
			ReserveX:          uint256.NewInt(111),
			ReserveY:          uint256.NewInt(222),
			LatestUpdateBlock: 11,
			ConcentrationK:    5001,
			StateUpdatedAt:    time.Now(),
			ReservesUpdatedAt: time.Now(),
			BlockNumber:       11,
		},
	}
	defer func() { subscriberInstance = nil }()

	syncData, err := coreABI.Events["Sync"].Inputs.Pack(big.NewInt(300), big.NewInt(400))
	if err != nil {
		t.Fatalf("pack sync event: %v", err)
	}

	tracker := NewPoolTracker(&Config{DexID: DexType, ChainID: valueobject.ChainIDBase}, nil)
	got, err := tracker.GetNewPoolState(context.Background(), poolEntity, pool.GetNewPoolStateParams{
		Logs: []types.Log{
			{
				Topics:      []common.Hash{topicSync},
				Data:        syncData,
				BlockNumber: 12,
			},
		},
	})
	if err != nil {
		t.Fatalf("get new pool state: %v", err)
	}

	if got.Reserves[0] != "300" || got.Reserves[1] != "400" {
		t.Fatalf("expected log reserves 300/400, got %s/%s", got.Reserves[0], got.Reserves[1])
	}
	if got.BlockNumber != 12 {
		t.Fatalf("expected block number 12, got %d", got.BlockNumber)
	}
}

func TestProcessLogsUpdatesLatestUpdateBlock(t *testing.T) {
	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:      uint256.NewInt(1),
		FeeAskX24:         0,
		FeeBidX24:         1,
		LatestUpdateBlock: 10,
		BlockDelay:        5,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	poolEntity := entity.Pool{
		Address:     "0x00003bf45ce34bf1bea78669f9a40ee630e11b99",
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 10,
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       string(extraBytes),
	}

	stateData, err := coreABI.Events["StateUpdated"].Inputs.Pack(
		big.NewInt(123), // anchorPrice (uint160)
		big.NewInt(456), // feeAskX24 (uint24)
		big.NewInt(789), // feeBidX24 (uint24)
	)
	if err != nil {
		t.Fatalf("pack state updated event: %v", err)
	}

	tracker := NewPoolTracker(&Config{DexID: DexType, ChainID: valueobject.ChainIDBase}, nil)
	got, err := tracker.processLogs(poolEntity, []types.Log{
		{
			Topics:      []common.Hash{topicStateUpdated},
			Data:        stateData,
			BlockNumber: 25,
		},
	})
	if err != nil {
		t.Fatalf("process logs: %v", err)
	}

	var extra Extra
	if err := json.Unmarshal([]byte(got.Extra), &extra); err != nil {
		t.Fatalf("unmarshal extra: %v", err)
	}

	if extra.LatestUpdateBlock != 25 {
		t.Fatalf("expected latest update block 25, got %d", extra.LatestUpdateBlock)
	}
	// p.BlockNumber must NOT be set to latestLogBlock when it equals
	// LatestUpdateBlock; that would make IsStale always return 0-diff.
	// The old value (10) is kept so subsequent blocks can advance past it.
	if got.BlockNumber != 10 {
		t.Fatalf("expected pool block number 10 (unchanged), got %d", got.BlockNumber)
	}
	if extra.SqrtPriceX96 == nil || extra.SqrtPriceX96.Uint64() != 123 {
		t.Fatalf("expected SqrtPriceX96 123, got %v", extra.SqrtPriceX96)
	}
	if extra.FeeAskX24 != 456 || extra.FeeBidX24 != 789 {
		t.Fatalf("expected fees 456/789, got %d/%d", extra.FeeAskX24, extra.FeeBidX24)
	}
}

// TestStateUpdatedLogDoesNotDisableStaleCheck is a regression for the bug where
// processing a StateUpdated-only log set p.BlockNumber == extra.LatestUpdateBlock,
// making IsStale always return false (0-diff) regardless of how many blocks passed.
func TestStateUpdatedLogDoesNotDisableStaleCheck(t *testing.T) {
	const blockDelay = 5

	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:      uint256.NewInt(1),
		LatestUpdateBlock: 10,
		BlockDelay:        blockDelay,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	poolEntity := entity.Pool{
		Address:     "0x00003bf45ce34bf1bea78669f9a40ee630e11b99",
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 10,
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       string(extraBytes),
	}

	stateData, err := coreABI.Events["StateUpdated"].Inputs.Pack(
		big.NewInt(999), // anchorPrice
		big.NewInt(0),   // feeAskX24
		big.NewInt(0),   // feeBidX24
	)
	if err != nil {
		t.Fatalf("pack state updated event: %v", err)
	}

	tracker := NewPoolTracker(&Config{DexID: DexType, ChainID: valueobject.ChainIDBase}, nil)

	// Simulate a StateUpdated at block 20 — the only event in this batch.
	got, err := tracker.processLogs(poolEntity, []types.Log{
		{Topics: []common.Hash{topicStateUpdated}, Data: stateData, BlockNumber: 20},
	})
	if err != nil {
		t.Fatalf("process logs: %v", err)
	}

	var extra Extra
	if err := json.Unmarshal([]byte(got.Extra), &extra); err != nil {
		t.Fatalf("unmarshal extra: %v", err)
	}
	if extra.LatestUpdateBlock != 20 {
		t.Fatalf("LatestUpdateBlock: want 20, got %d", extra.LatestUpdateBlock)
	}

	// p.BlockNumber must NOT equal LatestUpdateBlock; that collapses the
	// stale-check diff to 0 and prevents staleness detection permanently.
	if got.BlockNumber == extra.LatestUpdateBlock {
		t.Fatalf("p.BlockNumber (%d) == LatestUpdateBlock (%d): stale check is permanently disabled",
			got.BlockNumber, extra.LatestUpdateBlock)
	}

	// Now simulate a Sync log arriving blockDelay+1 blocks after the StateUpdated.
	// The pool is stale at that point and IsStale must return true.
	staleBlock := extra.LatestUpdateBlock + blockDelay + 1
	syncData, err := coreABI.Events["Sync"].Inputs.Pack(big.NewInt(300), big.NewInt(400))
	if err != nil {
		t.Fatalf("pack sync event: %v", err)
	}
	got2, err := tracker.processLogs(got, []types.Log{
		{Topics: []common.Hash{topicSync}, Data: syncData, BlockNumber: staleBlock},
	})
	if err != nil {
		t.Fatalf("process logs (sync): %v", err)
	}
	var extra2 Extra
	if err := json.Unmarshal([]byte(got2.Extra), &extra2); err != nil {
		t.Fatalf("unmarshal extra2: %v", err)
	}
	if !extra2.IsStale(got2.BlockNumber) {
		t.Fatalf("expected pool to be stale at block %d (LatestUpdateBlock=%d, BlockDelay=%d)",
			got2.BlockNumber, extra2.LatestUpdateBlock, extra2.BlockDelay)
	}
}
