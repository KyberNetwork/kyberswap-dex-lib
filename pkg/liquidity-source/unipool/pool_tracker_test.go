package unipool

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// NOTE: The tracker's main path (fetchState) hits an RPC node, so a full
// integration test requires a deployed UniPool instance. These tests cover the
// pieces that DON'T require network: instantiation and the Extra ↔ JSON ↔
// simulator round-trip the tracker is expected to produce.
//
// Live integration coverage will be added under //go:build integration once
// UniPool is deployed (cf. nabla / integral patterns in this repo).

func TestNewPoolTracker_Smoke(t *testing.T) {
	t.Parallel()
	tracker, err := NewPoolTracker(&Config{
		DexID:          "unipool",
		FactoryAddress: "0x1234567890123456789012345678901234567890",
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, tracker)
	assert.NotNil(t, tracker.config)
}

// TestTrackerProducedExtra_IsConsumedBySimulator verifies the contract between
// what the tracker serialises into entity.Pool.Extra and what NewPoolSimulator
// expects to read. The Extra struct is the only handover surface; if its
// shape ever drifts, this test will catch it.
func TestTrackerProducedExtra_IsConsumedBySimulator(t *testing.T) {
	t.Parallel()

	// Build the exact same Extra struct the tracker would serialise after a
	// successful fetchState (see pool_tracker.go GetNewPoolState).
	extra := Extra{
		Reserve0:              new(big.Int).SetUint64(1_000_000),
		Reserve1:              new(big.Int).SetUint64(2_000_000),
		VirtualReserve0In:     new(big.Int).SetUint64(1_100_000),
		VirtualReserve0Out:    new(big.Int).SetUint64(900_000),
		VirtualReserve1In:     new(big.Int).SetUint64(2_100_000),
		VirtualReserve1Out:    new(big.Int).SetUint64(1_900_000),
		LastUpdateTimestamp:   uint64(time.Now().Unix()),
		PriceDecay:            300,
		FeeLpBps:              25,
		FeePoolBps:            5,
		TotalBorrowed0:        new(big.Int).SetUint64(50_000),
		TotalBorrowed1:        new(big.Int).SetUint64(100_000),
		SwapPriceToleranceBps: 500,
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:  "0xcccccccccccccccccccccccccccccccccccccccc",
		Exchange: "unipool",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000", "2000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Swappable: true},
			{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: `{"factoryAddress":"0x1234567890123456789012345678901234567890"}`,
		BlockNumber: 12345,
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	require.NotNil(t, sim)

	// Spot-check that all dynamic fields landed in the simulator.
	assert.Equal(t, "1000000", sim.reserve0.Dec())
	assert.Equal(t, "2000000", sim.reserve1.Dec())
	assert.Equal(t, "1100000", sim.vr0In.Dec())
	assert.Equal(t, "900000", sim.vr0Out.Dec())
	assert.Equal(t, "2100000", sim.vr1In.Dec())
	assert.Equal(t, "1900000", sim.vr1Out.Dec())
	assert.Equal(t, extra.LastUpdateTimestamp, sim.lastUpdateTimestamp)
	assert.Equal(t, uint64(300), sim.priceDecay)
	assert.Equal(t, "25", sim.feeLpBps.Dec())
	assert.Equal(t, "5", sim.feePoolBps.Dec())
	assert.Equal(t, "50000", sim.totalBorrowed0.Dec())
	assert.Equal(t, "100000", sim.totalBorrowed1.Dec())
	assert.Equal(t, uint16(500), sim.swapPriceToleranceBps)
}

// zeroExtra (used by both factory and list-updater at discovery time) must
// also be consumable by the simulator without crashing — even though it
// represents a not-yet-tracked pool with all-zero state.
func TestZeroExtra_IsConsumedBySimulator(t *testing.T) {
	t.Parallel()
	extra := zeroExtra()
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address: "0xc0ffeec0ffeec0ffeec0ffeec0ffeec0ffeec0ff",
		Type:    DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    string(extraBytes),
	}
	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	require.NotNil(t, sim)
	assert.Equal(t, "0", sim.reserve0.Dec())
	assert.Equal(t, "0", sim.reserve1.Dec())
}
