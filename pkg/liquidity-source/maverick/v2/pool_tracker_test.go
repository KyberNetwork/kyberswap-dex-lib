package maverickv2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestGetFullPoolState(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	// Create ethrpc client
	ethrpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

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
	ethrpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

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
	ethrpcClient := ethrpc.New("https://ethereum-rpc.kyberswap.com").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

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

const multicallABIJSON = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"payable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

func packMaverickStage1Response(t *testing.T, blockNumber uint64) string {
	t.Helper()

	mcABI, err := abi.JSON(strings.NewReader(multicallABIJSON))
	require.NoError(t, err)

	type abiPoolState struct {
		ReserveA           *big.Int
		ReserveB           *big.Int
		LastTwaD8          int64
		LastLogPriceD8     int64
		LastTimestamp      *big.Int
		ActiveTick         int32
		IsLocked           bool
		BinCounter         uint32
		ProtocolFeeRatioD3 uint8
	}

	stateRet, err := maverickV2PoolABI.Methods[poolMethodGetState].Outputs.Pack(abiPoolState{
		ReserveA:           big.NewInt(5000),
		ReserveB:           big.NewInt(6000),
		LastTwaD8:          10,
		LastLogPriceD8:     12,
		LastTimestamp:      big.NewInt(1700000000),
		ActiveTick:         1,
		IsLocked:           false,
		BinCounter:         10,
		ProtocolFeeRatioD3: 5,
	})
	require.NoError(t, err)

	feeARet, err := maverickV2PoolABI.Methods["fee"].Outputs.Pack(big.NewInt(100))
	require.NoError(t, err)

	feeBRet, err := maverickV2PoolABI.Methods["fee"].Outputs.Pack(big.NewInt(200))
	require.NoError(t, err)

	type returnDataItem struct {
		Success    bool
		ReturnData []byte
	}
	packed, err := mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(
		new(big.Int).SetUint64(blockNumber),
		[32]byte{1},
		[]returnDataItem{
			{Success: true, ReturnData: stateRet},
			{Success: true, ReturnData: feeARet},
			{Success: true, ReturnData: feeBRet},
		},
	)
	require.NoError(t, err)

	return hexutil.Encode(packed)
}

func TestGetNewPoolState_Stage2Error(t *testing.T) {
	t.Parallel()

	stage1Hex := packMaverickStage1Response(t, 101)
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")

		if callCount.Add(1) == 1 {
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, stage1Hex)
			return
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"lens rpc error"}}`, req.ID)
	}))
	defer srv.Close()

	ethrpcClient := ethrpc.New(srv.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker, err := NewPoolTracker(&Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}, ethrpcClient)
	require.NoError(t, err)

	origExtra := `{"feeAIn":100,"feeBIn":200,"protocolFeeRatio":5,"bins":{"1":{"tick":1,"totalSupply":"0x64","tickBalance":"0x64"}},"ticks":{"1":{"ReserveA":"0x32","ReserveB":"0x32","TotalSupply":"0x64","BinIdsByTick":{"0":1}}},"activeTick":1,"lastTwaD8":10}`
	p := entity.Pool{
		Address:     "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca",
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 100,
		Reserves:    entity.PoolReserves{"1000", "2000"},
		Extra:       origExtra,
		StaticExtra: `{"tickSpacing":10}`,
	}

	got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.Error(t, err)
	assert.Equal(t, int32(2), callCount.Load())
	assert.Equal(t, origExtra, got.Extra)
}

func TestUpdatePool_ContextPropagatedToStage2(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"should not reach server when ctx canceled"}}`, req.ID)
	}))
	defer srv.Close()

	ethrpcClient := ethrpc.New(srv.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker, err := NewPoolTracker(&Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}, ethrpcClient)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	origExtra := `{"bins":{"1":{"tick":1,"totalSupply":"0x64","tickBalance":"0x64"}}}`
	p := entity.Pool{
		Address:     "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca",
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 100,
		Reserves:    entity.PoolReserves{"1000", "2000"},
		Extra:       origExtra,
	}
	state := State{
		ReserveA:   big.NewInt(5000),
		ReserveB:   big.NewInt(6000),
		BinCounter: 10,
	}

	got, err := tracker.updatePool(ctx, p, state, big.NewInt(101), nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), callCount.Load())
	assert.Equal(t, origExtra, got.Extra)
}
