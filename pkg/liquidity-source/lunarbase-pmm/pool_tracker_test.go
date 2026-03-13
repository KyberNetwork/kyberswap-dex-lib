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
		PX96:              uint256.NewInt(1),
		Fee:               1,
		LatestUpdateBlock: 10,
		BlockDelay:        5,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	poolEntity := entity.Pool{
		Address:     defaultCoreAddress,
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 10,
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       string(extraBytes),
	}

	subscriberInstance = &FlashBlockSubscriber{
		latestState: &poolState{
			PX96:              uint256.NewInt(2),
			FeeQ48:            2,
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
		PX96:              uint256.NewInt(1),
		Fee:               1,
		LatestUpdateBlock: 10,
		BlockDelay:        5,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	poolEntity := entity.Pool{
		Address:     defaultCoreAddress,
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 10,
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       string(extraBytes),
	}

	stateTuple := struct {
		PX96 *big.Int `abi:"pX96"`
		Fee  *big.Int `abi:"fee"`
	}{
		PX96: big.NewInt(123),
		Fee:  big.NewInt(456),
	}
	stateData, err := coreABI.Events["StateUpdated"].Inputs.Pack(stateTuple)
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
	if got.BlockNumber != 25 {
		t.Fatalf("expected pool block number 25, got %d", got.BlockNumber)
	}
}
