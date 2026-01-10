package maverickv2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGetFullPoolState(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	// Create ethrpc client
	ethrpcClient := ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Create pool tracker
	config := &Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}
	tracker, err := NewPoolTracker(config, ethrpcClient)
	assert.NoError(t, err)

	// Test parameters
	poolAddress := "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca"
	binCounter := uint32(615)

	// Get full pool state
	bins, ticks, err := tracker.getFullPoolState(context.Background(), poolAddress, binCounter, nil)
	assert.NoError(t, err)
	// print binId ascending
	for binId, bin := range bins {
		if binId == 16 {
			t.Logf("Bin %d: %+v", binId, bin)
		}
	}
	// t.Log(bins[16].ReserveA.String())
	// t.Log(bins[16].ReserveB.String())
	// t.Log(bins[16].TotalSupply.String())
	// t.Log(bins[16].CurrentLiquidity.String())
	// t.Log(bins[16].Tick)
	// t.Log(bins[16].TickBalance.String())
	// t.Log(bins[16].MergeBinBalance.String())

	// fmt.Println("bins", bins)
	// Log some basic info
	t.Logf("Number of bins: %d", len(bins))
	t.Logf("Number of ticks: %d", len(ticks))

	for tickId, tick := range ticks {
		if tickId == 4 {
			for _, binID := range tick.BinIdsByTick {
				if binID == 16 {
					t.Log(binReserves(bins[binID], tick))
				}
			}
		}

	}

	// Verify bin data
	for binId, bin := range bins {
		assert.NotNil(t, bin, "Bin %d should not be nil", binId)
		assert.NotNil(t, bin.TotalSupply, "Bin %d total supply should not be nil", binId)
	}
}

func TestGetState(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	// Create ethrpc client
	ethrpcClient := ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Create pool tracker
	config := &Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}
	tracker, err := NewPoolTracker(config, ethrpcClient)
	assert.NoError(t, err)

	// Test with multiple pool addresses
	testCases := []struct {
		name        string
		poolAddress string
		description string
	}{

		{
			name:        "USDC_USDT_Pool",
			poolAddress: "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca",
			description: "USDC/USDT pool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get pool state
			state, blockNumber, err := tracker.getState(context.Background(), tc.poolAddress, nil)
			assert.NoError(t, err, "getState should not return error for %s", tc.description)
			assert.NotNil(t, blockNumber, "Block number should not be nil")

			// Log basic info
			t.Logf("Pool: %s (%s)", tc.description, tc.poolAddress)
			t.Logf("Block number: %s", blockNumber.String())
			t.Logf("Reserve A: %s", state.ReserveA.String())
			t.Logf("Reserve B: %s", state.ReserveB.String())
			t.Logf("Active tick: %d", state.ActiveTick)
			t.Logf("Bin counter: %d", state.BinCounter)
			t.Logf("Fee A In: %d", state.FeeAIn)
			t.Logf("Fee B In: %d", state.FeeBIn)
			t.Logf("Protocol fee ratio: %d", state.ProtocolFeeRatioD3)
			t.Logf("Is locked: %t", state.IsLocked)
			t.Logf("Last TWA D8: %d", state.LastTwaD8)
			t.Logf("Last log price D8: %d", state.LastLogPriceD8)

			// Validate state fields
			assert.NotNil(t, state.ReserveA, "ReserveA should not be nil")
			assert.NotNil(t, state.ReserveB, "ReserveB should not be nil")
			assert.True(t, state.ReserveA.Cmp(big.NewInt(0)) >= 0, "ReserveA should be non-negative")
			assert.True(t, state.ReserveB.Cmp(big.NewInt(0)) >= 0, "ReserveB should be non-negative")
			assert.True(t, state.BinCounter > 0, "BinCounter should be positive")
			assert.True(t, blockNumber.Cmp(big.NewInt(0)) > 0, "Block number should be positive")

			// Validate fee values (should be reasonable percentages)
			// Fees are typically in basis points or similar small units
			// assert.True(t, state.FeeAIn >= 0, "FeeAIn should be non-negative")
			// assert.True(t, state.FeeBIn >= 0, "FeeBIn should be non-negative")
			// assert.True(t, state.ProtocolFeeRatioD3 >= 0, "ProtocolFeeRatioD3 should be non-negative")

			// Check that at least one reserve has liquidity (unless it's a completely empty pool)
			hasLiquidity := state.ReserveA.Cmp(big.NewInt(0)) > 0 || state.ReserveB.Cmp(big.NewInt(0)) > 0
			if !hasLiquidity {
				t.Logf("Warning: Pool %s appears to have no liquidity", tc.poolAddress)
			}
		})
	}
}

func TestGetFullPoolStateWithDifferentBatchSizes(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	// Create ethrpc client
	ethrpcClient := ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Test parameters
	poolAddress := "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca"

	// Create pool tracker to get binCounter
	config := &Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}
	tracker, err := NewPoolTracker(config, ethrpcClient)
	assert.NoError(t, err)

	// Get current state to get binCounter
	state, _, err := tracker.getState(context.Background(), poolAddress, nil)
	assert.NoError(t, err)
	binCounter := state.BinCounter

	// Test cases with different batch sizes
	testCases := []struct {
		name      string
		batchSize int
	}{
		{
			name:      "Default Batch Size",
			batchSize: DefaultBinBatchSize, // 500
		},
		{
			name:      "Small Batch Size",
			batchSize: 5,
		},
	}

	// Store results for comparison
	results := make(map[string]struct {
		bins  map[uint32]Bin
		ticks map[int32]Tick
	})

	// Run tests for each batch size
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create pool tracker with specific batch size
			config := &Config{
				PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
			}
			tracker, err := NewPoolTracker(config, ethrpcClient)
			assert.NoError(t, err)

			// Override default batch size
			DefaultBinBatchSize = tc.batchSize

			// Get full pool state
			bins, ticks, err := tracker.getFullPoolState(context.Background(), poolAddress, binCounter, nil)
			assert.NoError(t, err)

			// Store results for comparison
			results[tc.name] = struct {
				bins  map[uint32]Bin
				ticks map[int32]Tick
			}{
				bins:  bins,
				ticks: ticks,
			}

			// Log basic info
			t.Logf("Batch size: %d", tc.batchSize)
			t.Logf("Number of bins: %d", len(bins))
			t.Logf("Number of ticks: %d", len(ticks))
		})
	}

	// Compare results
	t.Run("Compare Results", func(t *testing.T) {
		defaultResult := results["Default Batch Size"]
		smallBatchResult := results["Small Batch Size"]

		// Compare bin counts
		assert.Equal(t, len(defaultResult.bins), len(smallBatchResult.bins),
			"Number of bins should be the same")

		// Compare tick counts
		assert.Equal(t, len(defaultResult.ticks), len(smallBatchResult.ticks),
			"Number of ticks should be the same")

		// Compare bin data
		for binId, defaultBin := range defaultResult.bins {
			smallBatchBin, exists := smallBatchResult.bins[binId]
			assert.True(t, exists, "Bin %d should exist in small batch results", binId)

			// Compare bin data
			if defaultBin.TotalSupply != nil {
				assert.Equal(t, defaultBin.TotalSupply.String(), smallBatchBin.TotalSupply.String(),
					"Bin %d TotalSupply should match", binId)
			}
			if defaultBin.TickBalance != nil {
				assert.Equal(t, defaultBin.TickBalance.String(), smallBatchBin.TickBalance.String(),
					"Bin %d TickBalance should match", binId)
			}
			assert.Equal(t, defaultBin.Tick, smallBatchBin.Tick,
				"Bin %d Tick should match", binId)
		}

		// Compare tick data
		for tickId, defaultTick := range defaultResult.ticks {
			smallBatchTick, exists := smallBatchResult.ticks[tickId]
			assert.True(t, exists, "Tick %d should exist in small batch results", tickId)

			// Compare tick data
			if defaultTick.TotalSupply != nil {
				assert.Equal(t, defaultTick.TotalSupply.String(), smallBatchTick.TotalSupply.String(),
					"Tick %d TotalSupply should match", tickId)
			}
			if defaultTick.ReserveA != nil {
				assert.Equal(t, defaultTick.ReserveA.String(), smallBatchTick.ReserveA.String(),
					"Tick %d ReserveA should match", tickId)
			}
			if defaultTick.ReserveB != nil {
				assert.Equal(t, defaultTick.ReserveB.String(), smallBatchTick.ReserveB.String(),
					"Tick %d ReserveB should match", tickId)
			}

			// Compare bin IDs in tick
			assert.Equal(t, len(defaultTick.BinIdsByTick), len(smallBatchTick.BinIdsByTick),
				"Tick %d should have same number of bins", tickId)
			for kind, binId := range defaultTick.BinIdsByTick {
				assert.Equal(t, binId, smallBatchTick.BinIdsByTick[kind],
					"Tick %d bin ID for kind %d should match", tickId, kind)
			}
		}
	})
}
