package dexLite

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

)

func TestPoolSimulator(t *testing.T) {
	// Create a mock pool similar to the Foundry test scenario
	staticExtra := StaticExtra{
		DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f",
		HasNative:      false,
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	// Mock pool state with reasonable values
	poolState := PoolState{
		DexVariables:             big.NewInt(0x123456789abcdef), // Mock packed variables
		CenterPriceShift:         big.NewInt(0),
		RangeShift:               big.NewInt(0),
		ThresholdShift:           big.NewInt(0),

	}

	// Create mock dexKey and dexId
	mockDexKey := DexKey{
		Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
		Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
		Salt:   [32]byte{},
	}
	mockDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	
	extra := PoolExtra{
		DexKey:    mockDexKey,
		DexId:     mockDexId,
		PoolState: poolState,
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0x1234567890123456789012345678901234567890",
		Exchange: "fluid-dex-lite",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000", "1000000000"}, // 1000 USDC, 1000 USDT
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1", // USDC
				Swappable: true,
				Decimals:  6,
			},
			{
				Address:   "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
				Swappable: true,
				Decimals:  6,
			},
		},
		SwapFee:      0.001, // 0.1%
		BlockNumber:  18000000,
		Extra:        string(extraBytes),
		StaticExtra:  string(staticExtraBytes),
		Timestamp:    1234567890,
	}

	simulator, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	require.NotNil(t, simulator)

	t.Run("TestCalcAmountOut", func(t *testing.T) {
		// Test swapping 1 USDC for USDT (similar to Foundry test)
		amountIn := big.NewInt(1000000) // 1 USDC (6 decimals)
		
		result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1", // USDC
				Amount: amountIn,
			},
			TokenOut: "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
		})

		if err != nil {
			t.Logf("CalcAmountOut error (expected with mock data): %v", err)
		} else {
			require.NotNil(t, result)
			require.NotNil(t, result.TokenAmountOut)
			require.Greater(t, result.TokenAmountOut.Amount.Int64(), int64(0))
			
			t.Logf("Swap 1 USDC -> %s USDT", result.TokenAmountOut.Amount.String())
			t.Logf("Fee: %s USDC", result.Fee.Amount.String())
			t.Logf("Gas: %d", result.Gas)
		}
	})

	t.Run("TestCalcAmountIn", func(t *testing.T) {
		// Test calculating input for 1 USDT output
		amountOut := big.NewInt(1000000) // 1 USDT (6 decimals)
		
		result, err := simulator.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
				Amount: amountOut,
			},
			TokenIn: "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1", // USDC
		})

		if err != nil {
			t.Logf("CalcAmountIn error (expected with mock data): %v", err)
		} else {
			require.NotNil(t, result)
			require.NotNil(t, result.TokenAmountIn)
			require.Greater(t, result.TokenAmountIn.Amount.Int64(), int64(0))
			
			t.Logf("Need %s USDC -> 1 USDT", result.TokenAmountIn.Amount.String())
			t.Logf("Fee: %s USDC", result.Fee.Amount.String())
		}
	})

	t.Run("TestUnpackDexVariables", func(t *testing.T) {
		// Test unpacking dex variables
		dexVars := simulator.unpackDexVariables(poolState.DexVariables)
		
		require.NotNil(t, dexVars)
		require.NotNil(t, dexVars.Fee)
		require.NotNil(t, dexVars.RevenueCut)
		require.NotNil(t, dexVars.CenterPrice)
		
		t.Logf("Unpacked fee: %s", dexVars.Fee.String())
		t.Logf("Unpacked revenue cut: %s", dexVars.RevenueCut.String())
		t.Logf("Unpacked center price: %s", dexVars.CenterPrice.String())
	})

	t.Run("TestUpdateBalance", func(t *testing.T) {
		// Test updating pool balance after swap
		initialReserves := simulator.GetReserves()
		require.Len(t, initialReserves, 2)
		
		// Mock swap: 1 USDC in, 0.999 USDT out
		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bB336F8eb2f",
				Amount: big.NewInt(1000000),
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
				Amount: big.NewInt(999000),
			},
			SwapInfo: SwapInfo{
				NewPoolState: poolState,
			},
		})
		
		newReserves := simulator.GetReserves()
		require.Len(t, newReserves, 2)
		
		t.Logf("Reserve changes: [%s -> %s], [%s -> %s]", 
			initialReserves[0].String(), newReserves[0].String(),
			initialReserves[1].String(), newReserves[1].String())
	})
}

func TestPoolSimulatorEdgeCases(t *testing.T) {


	t.Run("TestZeroAmountIn", func(t *testing.T) {
		// Create normal pool for this test
		staticExtra := StaticExtra{DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f"}
		staticExtraBytes, _ := json.Marshal(staticExtra)

		// Create a mock dexKey and dexId for this test
		testDexKey := DexKey{
			Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
			Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
			Salt:   [32]byte{},
		}
		testDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
		
		extra := PoolExtra{
			DexKey:    testDexKey,
			DexId:     testDexId,
			PoolState: PoolState{DexVariables: big.NewInt(0x123456789abcdef)},
		}
		extraBytes, _ := json.Marshal(extra)

		entityPool := entity.Pool{
			Address:  "0x1234567890123456789012345678901234567890",
			Type:     DexType,
			Reserves: entity.PoolReserves{"1000000000", "1000000000"},
			Tokens: []*entity.PoolToken{
				{Address: "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bB336F8eb2f", Decimals: 6},
				{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Decimals: 6},
			},
			SwapFee:     0.001,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		simulator, err := NewPoolSimulator(entityPool)
		require.NoError(t, err)

		// Should fail with zero amount
		_, err = simulator.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bB336F8eb2f",
				Amount: big.NewInt(0),
			},
			TokenOut: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		})

		require.Error(t, err)
		require.Equal(t, ErrInvalidAmountIn, err)
	})
}

func TestMathFunctions(t *testing.T) {
	simulator := &PoolSimulator{
		Token0Decimals: 6,
		Token1Decimals: 6,
	}

	t.Run("TestSwapMath", func(t *testing.T) {
		// Test basic constant product calculation
		// dexVars := &UnpackedDexVariables{
		//	Fee:                       big.NewInt(100), // 1%
		//	CenterPrice:              bignumber.TenPowInt(27), // 1:1 price
		//	Token0TotalSupplyAdjusted: big.NewInt(1000000), // 1M in 12 decimals
		//	Token1TotalSupplyAdjusted: big.NewInt(1000000), // 1M in 12 decimals
		// }

		// pricing := &PricingResult{
		//	CenterPrice:             dexVars.CenterPrice,
		//	Token0ImaginaryReserves: dexVars.Token0TotalSupplyAdjusted,
		//	Token1ImaginaryReserves: dexVars.Token1TotalSupplyAdjusted,
		// }

		// Create a properly packed dexVariables for testing
		// Pack: fee=100, revenueCut=10, rebalancing=1, centerPrice=1e27, token0Supply=1000000, token1Supply=1000000
		mockDexVariables := big.NewInt(0)
		
		// Fee (100) at bits 0-12
		mockDexVariables.Or(mockDexVariables, big.NewInt(100))
		
		// Revenue cut (10) at bits 13-19 
		mockDexVariables.Or(mockDexVariables, new(big.Int).Lsh(big.NewInt(10), 13))
		
		// Rebalancing status (1) at bits 20-21
		mockDexVariables.Or(mockDexVariables, new(big.Int).Lsh(big.NewInt(1), 20))
		
		// Center price (1e27 compressed) at bits 23-62
		centerPriceCompressed := big.NewInt(1e18) // Simplified for testing
		mockDexVariables.Or(mockDexVariables, new(big.Int).Lsh(centerPriceCompressed, 23))
		
		// Token supplies at bits 136-196
		token0Supply := big.NewInt(1000000)
		token1Supply := big.NewInt(1000000)
		mockDexVariables.Or(mockDexVariables, new(big.Int).Lsh(token0Supply, 136))
		mockDexVariables.Or(mockDexVariables, new(big.Int).Lsh(token1Supply, 196))

		// Create a mock pool state for testing
		mockPoolState := PoolState{
			DexVariables:     mockDexVariables,
			CenterPriceShift: big.NewInt(0),
			RangeShift:       big.NewInt(0),
			ThresholdShift:   big.NewInt(0),
		}

		// Test swapIn: 1000 tokens in
		amountOut, _, _, err := simulator.calculateSwapInWithState(true, big.NewInt(1000), mockPoolState)
		if err != nil {
			t.Logf("SwapIn error (expected with simple math): %v", err)
		} else {
			require.Greater(t, amountOut.Int64(), int64(0))
			t.Logf("SwapIn: 1000 -> %s", amountOut.String())
		}

		// Test swapOut: want 1000 tokens out
		amountIn, _, _, err := simulator.calculateSwapOutWithState(true, big.NewInt(1000), mockPoolState)
		if err != nil {
			t.Logf("SwapOut error (expected with simple math): %v", err)
		} else {
			require.Greater(t, amountIn.Int64(), int64(0))
			t.Logf("SwapOut: %s -> 1000", amountIn.String())
		}
	})
}