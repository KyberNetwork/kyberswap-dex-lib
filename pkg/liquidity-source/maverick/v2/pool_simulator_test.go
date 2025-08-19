package maverickv2

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func BenchmarkPoolSimulator_CalcAmountOut(b *testing.B) {
	// Read pool_data.json
	data, err := os.ReadFile("./data/pool_data.json")
	require.NoError(b, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(b, err)

	poolSim, _ := NewPoolSimulator(poolEntity)

	testutil.TestCalcAmountIn(b, poolSim)
}

func TestSimpleSwaps_USDC_USDT(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data

	// Read pool_data.json
	data, err := os.ReadFile("./data/pool_data.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Perform a swap to get swap info
	amountIn := big.NewInt(10000000) // 10 USDC
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // USDC
			Amount: amountIn,
		},
		TokenOut: poolEntity.Tokens[1].Address, // USDT
		Limit:    nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Extract swap info
	swapInfo, ok := result.SwapInfo.(maverickSwapInfo)
	require.True(t, ok)

	// Store state before UpdateBalance
	beforeUpdateReserveA := new(big.Int).Set(poolSim.Info.Reserves[0])
	beforeUpdateReserveB := new(big.Int).Set(poolSim.Info.Reserves[1])

	// Test UpdateBalance
	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // USDC
			Amount: amountIn,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  poolEntity.Tokens[1].Address, // USDT
			Amount: result.TokenAmountOut.Amount,
		},
		SwapInfo: swapInfo,
	})

	// Verify state updates
	t.Run("Active Tick Updated", func(t *testing.T) {
		require.Equal(t, swapInfo.activeTick, poolSim.state.ActiveTick, "Active tick should be updated from swap info")
	})

	t.Run("Bins Updated", func(t *testing.T) {
		require.Equal(t, len(swapInfo.bins), len(poolSim.state.Bins), "Bins count should match swap info")
		// Verify specific bins are updated
		for binId, expectedBin := range swapInfo.bins {
			actualBin, exists := poolSim.state.Bins[binId]
			require.True(t, exists, "Bin %d should exist", binId)
			require.Equal(t, expectedBin.TotalSupply.String(), actualBin.TotalSupply.String(), "Bin %d total supply should match", binId)
			require.Equal(t, expectedBin.TickBalance.String(), actualBin.TickBalance.String(), "Bin %d tick balance should match", binId)
		}
	})

	t.Run("Ticks Updated", func(t *testing.T) {
		require.Equal(t, len(swapInfo.ticks), len(poolSim.state.Ticks), "Ticks count should match swap info")
		// Verify specific ticks are updated
		for tickId, expectedTick := range swapInfo.ticks {
			actualTick, exists := poolSim.state.Ticks[tickId]
			require.True(t, exists, "Tick %d should exist", tickId)
			require.Equal(t, expectedTick.ReserveA.String(), actualTick.ReserveA.String(), "Tick %d reserve A should match", tickId)
			require.Equal(t, expectedTick.ReserveB.String(), actualTick.ReserveB.String(), "Tick %d reserve B should match", tickId)
		}
	})

	t.Run("Pool Reserves Updated", func(t *testing.T) {
		expectedReserveA := new(big.Int).Add(beforeUpdateReserveA, amountIn)
		expectedReserveB := new(big.Int).Sub(beforeUpdateReserveB, result.TokenAmountOut.Amount)

		require.Equal(t, expectedReserveA.String(), poolSim.Pool.Info.Reserves[0].String(), "Reserve A should be updated correctly")
		require.Equal(t, expectedReserveB.String(), poolSim.Pool.Info.Reserves[1].String(), "Reserve B should be updated correctly")
	})

	// Test error handling
	t.Run("Invalid SwapInfo Type", func(t *testing.T) {
		// This should not panic but should log a warning
		poolSim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  poolEntity.Tokens[0].Address,
				Amount: big.NewInt(1000),
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  poolEntity.Tokens[1].Address,
				Amount: big.NewInt(1000),
			},
			SwapInfo: "invalid_swap_info", // Wrong type
		})
		// Should not crash - just verify simulator is still functional
		require.NotNil(t, poolSim.state)
	})

	t.Run("CalcAmountIn", func(t *testing.T) {
		testutil.TestCalcAmountIn(t, poolSim, 2) // so slow...
	})
}

func TestSimpleSwaps_MAV_WETH(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data

	// Read pool_data.json
	data, err := os.ReadFile("./data/mavweth.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Perform a swap to get swap info
	amountIn := big.NewInt(1_000_000_000_000_000_000) // 1 MAV
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // MAV
			Amount: amountIn,
		},
		TokenOut: poolEntity.Tokens[1].Address, // WETH
		Limit:    nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert the expected amount
	expectedAmount := big.NewInt(22034312672685)
	require.Equal(t, expectedAmount.String(), result.TokenAmountOut.Amount.String(), "Swap amount should match expected value")

	t.Run("CalcAmountIn", func(t *testing.T) {
		testutil.TestCalcAmountIn(t, poolSim, 2) // so slow...
	})
}

func TestDebugSwap_MAV_WETH(t *testing.T) {
	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Read pool_data.json
	data, err := os.ReadFile("./data/mavweth.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Print pool information
	fmt.Printf("Pool Address: %s\n", poolEntity.Address)
	fmt.Printf("Token A (MAV): %s, decimals: %d\n", poolEntity.Tokens[0].Address, poolEntity.Tokens[0].Decimals)
	fmt.Printf("Token B (WETH): %s, decimals: %d\n", poolEntity.Tokens[1].Address, poolEntity.Tokens[1].Decimals)
	fmt.Printf("Reserve A (MAV): %s\n", poolEntity.Reserves[0])
	fmt.Printf("Reserve B (WETH): %s\n", poolEntity.Reserves[1])
	fmt.Printf("Fee A In: %d\n", poolSim.state.FeeAIn)
	fmt.Printf("Fee B In: %d\n", poolSim.state.FeeBIn)
	fmt.Printf("Protocol Fee Ratio: %d\n", poolSim.state.ProtocolFeeRatio)
	fmt.Printf("Active Tick: %d\n", poolSim.state.ActiveTick)
	// Debug tick information around active tick
	fmt.Printf("\nTick information around active tick %d:\n", poolSim.state.ActiveTick)
	for tick := poolSim.state.ActiveTick - 2; tick <= poolSim.state.ActiveTick+2; tick++ {
		if tickData, ok := poolSim.state.Ticks[tick]; ok {
			fmt.Printf("Tick %d: ReserveA=%s, ReserveB=%s, TotalSupply=%s\n",
				tick, tickData.ReserveA.String(), tickData.ReserveB.String(), tickData.TotalSupply.String())
		} else {
			fmt.Printf("Tick %d: No data\n", tick)
		}
	}
	// Debug bin information
	fmt.Printf("\nBin information:\n")
	binCount := 0
	for binId, bin := range poolSim.state.Bins {
		if binCount < 10 { // Show first 10 bins
			fmt.Printf("Bin %d: tick=%d, kind=%d, totalSupply=%s, tickBalance=%s\n",
				binId, bin.Tick, bin.Kind, bin.TotalSupply.String(), bin.TickBalance.String())
		}
		binCount++
	}
	fmt.Printf("Total bins: %d\n", binCount)

	// Test scaling functions
	fmt.Printf("\nTesting scaling functions:\n")
	scaleA := getScale(poolEntity.Tokens[0].Decimals)
	scaleB := getScale(poolEntity.Tokens[1].Decimals)
	fmt.Printf("Scale A (MAV, 18 decimals): %s\n", scaleA.String())
	fmt.Printf("Scale B (WETH, 18 decimals): %s\n", scaleB.String())

	// Perform a swap to get swap info
	amountIn, ok := new(big.Int).SetString("20000000000000000000", 10) // 20 MAV
	require.True(t, ok)
	fmt.Printf("\nSwapping 20 MAV (%s) for WETH\n", amountIn.String())

	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // MAV
			Amount: amountIn,
		},
		TokenOut: poolEntity.Tokens[1].Address, // WETH
		Limit:    nil,
	})

	if err != nil {
		fmt.Printf("Error during calculation: %v\n", err)
		require.NoError(t, err)
	}

	require.NotNil(t, result)

	fmt.Printf("Result amount: %s WETH\n", result.TokenAmountOut.Amount.String())
	fmt.Printf("Expected amount: 440665698723087 WETH\n")
	fmt.Printf("Current amount: %s WETH\n", result.TokenAmountOut.Amount.String())
	fmt.Printf("Gas estimation: %d\n", result.Gas)

	// Check if the result matches expected value
	expectedAmount := big.NewInt(440665698723087)
	if result.TokenAmountOut.Amount.Cmp(expectedAmount) != 0 {
		fmt.Printf("❌ MISMATCH: Expected %s, got %s\n", expectedAmount.String(), result.TokenAmountOut.Amount.String())
		fmt.Printf("Difference: %s\n", new(big.Int).Sub(result.TokenAmountOut.Amount, expectedAmount).String())

		// Calculate percentage difference
		diff := new(big.Int).Sub(result.TokenAmountOut.Amount, expectedAmount)
		percentage := new(big.Int).Mul(diff, big.NewInt(10000))
		percentage.Div(percentage, expectedAmount)
		fmt.Printf("Percentage difference: %s.%02d%%\n",
			new(big.Int).Div(percentage, big.NewInt(100)).String(),
			new(big.Int).Mod(percentage, big.NewInt(100)).Int64())
	} else {
		fmt.Printf("✅ MATCH: Result matches expected value\n")
	}
}
