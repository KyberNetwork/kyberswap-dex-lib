package gateway

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolSimulator_CalcAmountOut_AllPaths(t *testing.T) {
	// Actual token addresses from ethereum.json
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	iusdAddr := "0x48f9e38f3070ad8945dfae3fa70987722e3d89c"
	siusdAddr := "0xdbdc1ef57537e34680b898e1febd3d68c7389bcb"
	liusd1moAddr := "0x12b004719fb632f1e7c010c6f5d6009fb4258442"
	liusd2moAddr := "0xf1839becaf586814d022f16cdb3504ff8d8ff361"
	gatewayAddr := "0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5"

	testCases := []struct {
		name           string
		poolExtra      Extra
		tokenIn        string
		tokenOut       string
		amountIn       string
		expectedAmount string
		expectedError  error
		expectedGas    int64
	}{
		// Test 1: USDC → iUSD (mint)
		{
			name: "USDC -> iUSD (mint) - 1 USDC to 1 iUSD",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"), // 1M iUSD
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"), // 1M iUSD in vault
				SIUSDSupply:        mustParseBig("500000000000000000000000"),  // 500k siUSD
				LIUSDSupplies:      []string{"1000000000000000000000000"},     // 1M liUSD
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},     // 1M iUSD locked
			},
			tokenIn:        usdcAddr,
			tokenOut:       iusdAddr,
			amountIn:       "1000000",             // 1 USDC (6 decimals)
			expectedAmount: "1000000000000000000", // 1 iUSD (18 decimals)
			expectedError:  nil,
			expectedGas:    defaultMintGas,
		},

		// Test 2: iUSD → USDC (redeem)
		{
			name: "iUSD -> USDC (redeem) - 1 iUSD to 1 USDC",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000"},
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},
			},
			tokenIn:        iusdAddr,
			tokenOut:       usdcAddr,
			amountIn:       "1000000000000000000", // 1 iUSD (18 decimals)
			expectedAmount: "1000000",             // 1 USDC (6 decimals)
			expectedError:  nil,
			expectedGas:    defaultRedeemGas,
		},

		// Test 3: iUSD → siUSD (stake) with 2:1 exchange rate
		{
			name: "iUSD -> siUSD (stake) - 1 iUSD to 0.5 siUSD (2:1 rate)",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"), // 1M iUSD in vault
				SIUSDSupply:        mustParseBig("500000000000000000000000"),  // 500k siUSD (2:1 ratio)
				LIUSDSupplies:      []string{"1000000000000000000000000"},
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},
			},
			tokenIn:        iusdAddr,
			tokenOut:       siusdAddr,
			amountIn:       "1000000000000000000", // 1 iUSD
			expectedAmount: "500000000000000000",  // 0.5 siUSD (1 * 500k / 1M)
			expectedError:  nil,
			expectedGas:    defaultStakeGas,
		},

		// Test 4: siUSD → iUSD (unstake) with 2:1 exchange rate
		{
			name: "siUSD -> iUSD (unstake) - 1 siUSD to 2 iUSD (2:1 rate)",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"), // 1M iUSD in vault
				SIUSDSupply:        mustParseBig("500000000000000000000000"),  // 500k siUSD (2:1 ratio)
				LIUSDSupplies:      []string{"1000000000000000000000000"},
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},
			},
			tokenIn:        siusdAddr,
			tokenOut:       iusdAddr,
			amountIn:       "1000000000000000000", // 1 siUSD
			expectedAmount: "2000000000000000000", // 2 iUSD (1 * 1M / 500k)
			expectedError:  nil,
			expectedGas:    defaultUnstakeGas,
		},

		// Test 5: iUSD → liUSD-1mo (lock) with 1:1 ratio
		{
			name: "iUSD -> liUSD-1mo (lock) - 1 iUSD to 1 liUSD",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000"}, // 1M liUSD shares
				LIUSDTotalReceipts: []string{"1000000000000000000000000"}, // 1M iUSD locked (1:1 ratio)
			},
			tokenIn:        iusdAddr,
			tokenOut:       liusd1moAddr,
			amountIn:       "1000000000000000000", // 1 iUSD
			expectedAmount: "1000000000000000000", // 1 liUSD (1:1)
			expectedError:  nil,
			expectedGas:    defaultCreatePositionGas,
		},

		// Test 6: iUSD → liUSD-2mo (lock) with 0.8:1 ratio
		{
			name: "iUSD -> liUSD-2mo (lock) - 1 iUSD to 0.8 liUSD (0.8:1 rate)",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000", "800000000000000000000000"},  // 1M, 800k
				LIUSDTotalReceipts: []string{"1000000000000000000000000", "1000000000000000000000000"}, // 1M, 1M (0.8:1 ratio for bucket 2)
			},
			tokenIn:  iusdAddr,
			tokenOut: liusd2moAddr,
			amountIn: "1000000000000000000", // 1 iUSD
			// 1 iUSD * 800k shares / 1M receipts = 0.8 liUSD
			expectedAmount: "800000000000000000",
			expectedError:  nil,
			expectedGas:    defaultCreatePositionGas,
		},

		// Test 7: Contract paused
		{
			name: "Contract paused should fail",
			poolExtra: Extra{
				IsPaused:           true,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{},
				LIUSDTotalReceipts: []string{},
			},
			tokenIn:       usdcAddr,
			tokenOut:      iusdAddr,
			amountIn:      "1000000",
			expectedError: ErrContractPaused,
			expectedGas:   0,
		},

		// Test 8: Unsupported swap (USDC -> siUSD direct, skip iUSD)
		{
			name: "Unsupported swap (USDC -> siUSD direct) should fail",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{},
				LIUSDTotalReceipts: []string{},
			},
			tokenIn:       usdcAddr,
			tokenOut:      siusdAddr,
			amountIn:      "1000000",
			expectedError: ErrSwapNotSupported,
			expectedGas:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create pool entity with proper token list
			tokens := []*entity.PoolToken{
				{Address: usdcAddr, Decimals: 6, Swappable: true},
				{Address: iusdAddr, Decimals: 18, Swappable: true},
				{Address: siusdAddr, Decimals: 18, Swappable: true},
			}

			// Add liUSD tokens based on test case
			if len(tc.poolExtra.LIUSDSupplies) > 0 {
				tokens = append(tokens, &entity.PoolToken{Address: liusd1moAddr, Decimals: 18, Swappable: true})
			}
			if len(tc.poolExtra.LIUSDSupplies) > 1 {
				tokens = append(tokens, &entity.PoolToken{Address: liusd2moAddr, Decimals: 18, Swappable: true})
			}

			// Build reserves
			reserves := []string{
				tc.poolExtra.SIUSDTotalAssets.String(),
				tc.poolExtra.SIUSDSupply.String(),
			}
			reserves = append(reserves, tc.poolExtra.LIUSDSupplies...)

			poolEntity := entity.Pool{
				Address:  gatewayAddr,
				Exchange: "infinifi",
				Type:     DexType,
				Tokens:   tokens,
				Reserves: reserves,
			}

			// Marshal extra
			extraBytes, err := json.Marshal(tc.poolExtra)
			require.NoError(t, err)
			poolEntity.Extra = string(extraBytes)

			// Create simulator
			simulator, err := NewPoolSimulator(poolEntity)
			require.NoError(t, err)

			// Calculate amount out
			amountIn := mustParseBig(tc.amountIn)
			result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: amountIn,
				},
				TokenOut: tc.tokenOut,
			})

			// Check error
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedAmount, result.TokenAmountOut.Amount.String())
			if tc.expectedGas > 0 {
				assert.Equal(t, tc.expectedGas, result.Gas)
			}
			assert.Equal(t, big.NewInt(0), result.Fee.Amount) // No fees
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	// Actual token addresses from ethereum.json
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	iusdAddr := "0x48f9e38f3070ad8945dfae3fa70987722e3d89c"
	siusdAddr := "0xdbdc1ef57537e34680b898e1febd3d68c7389bcb"
	liusd1moAddr := "0x12b004719fb632f1e7c010c6f5d6009fb4258442"
	gatewayAddr := "0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5"

	initialIUSD := mustParseBig("1000000000000000000000000")
	initialSIUSDAssets := mustParseBig("1000000000000000000000000")
	initialSIUSDSupply := mustParseBig("500000000000000000000000")
	initialLIUSD1moSupply := mustParseBig("1000000000000000000000000")
	initialLIUSD1moReceipts := mustParseBig("1000000000000000000000000")

	poolExtra := Extra{
		IsPaused:           false,
		IUSDSupply:         initialIUSD,
		SIUSDTotalAssets:   initialSIUSDAssets,
		SIUSDSupply:        initialSIUSDSupply,
		LIUSDSupplies:      []string{initialLIUSD1moSupply.String()},
		LIUSDTotalReceipts: []string{initialLIUSD1moReceipts.String()},
	}

	poolEntity := entity.Pool{
		Address:  gatewayAddr,
		Exchange: "infinifi",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Decimals: 6, Swappable: true},
			{Address: iusdAddr, Decimals: 18, Swappable: true},
			{Address: siusdAddr, Decimals: 18, Swappable: true},
			{Address: liusd1moAddr, Decimals: 18, Swappable: true},
		},
		Reserves: []string{
			poolExtra.SIUSDTotalAssets.String(),
			poolExtra.SIUSDSupply.String(),
			poolExtra.LIUSDSupplies[0],
		},
	}

	extraBytes, err := json.Marshal(poolExtra)
	require.NoError(t, err)
	poolEntity.Extra = string(extraBytes)

	// Test 1: USDC → iUSD (mint)
	t.Run("USDC -> iUSD (mint)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInUSDC := mustParseBig("1000000")              // 1 USDC
		amountOutIUSD := mustParseBig("1000000000000000000") // 1 iUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: usdcAddr, Amount: amountInUSDC},
			TokenAmountOut: pool.TokenAmount{Token: iusdAddr, Amount: amountOutIUSD},
			Fee:            pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(0)},
		})

		expectedIUSD := new(big.Int).Add(initialIUSD, amountOutIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.iusdSupply.String())
	})

	// Test 2: iUSD → USDC (redeem)
	t.Run("iUSD -> USDC (redeem)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("1000000000000000000") // 1 iUSD
		amountOutUSDC := mustParseBig("1000000")            // 1 USDC

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: usdcAddr, Amount: amountOutUSDC},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedIUSD := new(big.Int).Sub(initialIUSD, amountInIUSD)
		assert.Equal(t, expectedIUSD.String(), simulator.iusdSupply.String())
	})

	// Test 3: iUSD → siUSD (stake)
	t.Run("iUSD -> siUSD (stake)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("1000000000000000000")  // 1 iUSD
		amountOutSIUSD := mustParseBig("500000000000000000") // 0.5 siUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: siusdAddr, Amount: amountOutSIUSD},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedSIUSDAssets := new(big.Int).Add(initialSIUSDAssets, amountInIUSD)
		expectedSIUSDSupply := new(big.Int).Add(initialSIUSDSupply, amountOutSIUSD)

		assert.Equal(t, expectedSIUSDAssets.String(), simulator.siusdTotalAssets.String())
		assert.Equal(t, expectedSIUSDSupply.String(), simulator.siusdSupply.String())
	})

	// Test 4: siUSD → iUSD (unstake)
	t.Run("siUSD -> iUSD (unstake)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInSIUSD := mustParseBig("1000000000000000000") // 1 siUSD
		amountOutIUSD := mustParseBig("2000000000000000000") // 2 iUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: siusdAddr, Amount: amountInSIUSD},
			TokenAmountOut: pool.TokenAmount{Token: iusdAddr, Amount: amountOutIUSD},
			Fee:            pool.TokenAmount{Token: siusdAddr, Amount: big.NewInt(0)},
		})

		expectedSIUSDAssets := new(big.Int).Sub(initialSIUSDAssets, amountOutIUSD)
		expectedSIUSDSupply := new(big.Int).Sub(initialSIUSDSupply, amountInSIUSD)

		assert.Equal(t, expectedSIUSDAssets.String(), simulator.siusdTotalAssets.String())
		assert.Equal(t, expectedSIUSDSupply.String(), simulator.siusdSupply.String())
	})

	// Test 5: iUSD → liUSD (lock)
	t.Run("iUSD -> liUSD (lock)", func(t *testing.T) {
		simulator, err := NewPoolSimulator(poolEntity)
		require.NoError(t, err)

		amountInIUSD := mustParseBig("2000000000000000000")   // 2 iUSD
		amountOutLIUSD := mustParseBig("2000000000000000000") // 2 liUSD

		simulator.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: iusdAddr, Amount: amountInIUSD},
			TokenAmountOut: pool.TokenAmount{Token: liusd1moAddr, Amount: amountOutLIUSD},
			Fee:            pool.TokenAmount{Token: iusdAddr, Amount: big.NewInt(0)},
		})

		expectedLIUSD1moSupply := new(big.Int).Add(initialLIUSD1moSupply, amountOutLIUSD)
		expectedLIUSD1moReceipts := new(big.Int).Add(initialLIUSD1moReceipts, amountInIUSD)

		assert.Equal(t, expectedLIUSD1moSupply.String(), simulator.liusdSupplies[0].String())
		assert.Equal(t, expectedLIUSD1moReceipts.String(), simulator.liusdTotalReceipts[0].String())
	})
}

func mustParseBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("failed to parse big.Int: " + s)
	}
	return b
}
