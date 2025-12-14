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

func TestPoolSimulator_CalcAmountOut_OneWay(t *testing.T) {
	// Actual token addresses from ethereum.json
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
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
		{
			name: "USDC -> siUSD (mintAndStake) - 1 USDC with 2:1 exchange rate",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"), // 1M iUSD
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"), // 1M iUSD in vault
				SIUSDSupply:        mustParseBig("500000000000000000000000"),  // 500k siUSD (2:1 ratio)
				LIUSDSupplies:      []string{"1000000000000000000000000"},     // 1M liUSD-1mo shares
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},     // 1M iUSD locked
			},
			tokenIn:        usdcAddr,
			tokenOut:       siusdAddr,
			amountIn:       "1000000",            // 1 USDC (6 decimals)
			expectedAmount: "500000000000000000", // 0.5 siUSD (18 decimals) - 1 iUSD * 500k / 1M
			expectedError:  nil,
			expectedGas:    defaultMintAndStakeGas,
		},
		{
			name: "USDC -> liUSD-1mo (mintAndLock) - 1 USDC to 1 liUSD",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000"}, // 1M liUSD-1mo shares
				LIUSDTotalReceipts: []string{"1000000000000000000000000"}, // 1M iUSD locked (1:1 ratio)
			},
			tokenIn:        usdcAddr,
			tokenOut:       liusd1moAddr,
			amountIn:       "1000000",             // 1 USDC
			expectedAmount: "1000000000000000000", // 1 liUSD (1:1)
			expectedError:  nil,
			expectedGas:    defaultMintAndLockGas,
		},
		{
			name: "USDC -> liUSD-2mo (mintAndLock) - with non-1:1 exchange rate",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000", "800000000000000000000000"},  // 1M, 800k
				LIUSDTotalReceipts: []string{"1000000000000000000000000", "1000000000000000000000000"}, // 1M, 1M (0.8:1 ratio)
			},
			tokenIn:  usdcAddr,
			tokenOut: liusd2moAddr,
			amountIn: "1000000", // 1 USDC
			// 1 USDC → 1e18 iUSD, then 1e18 * 800k / 1M = 0.8e18 liUSD
			expectedAmount: "800000000000000000",
			expectedError:  nil,
			expectedGas:    defaultMintAndLockGas,
		},
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
			tokenOut:      siusdAddr,
			amountIn:      "1000000",
			expectedError: ErrContractPaused,
			expectedGas:   0,
		},
		{
			name: "Unsupported swap (siUSD -> USDC) should fail",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{},
				LIUSDTotalReceipts: []string{},
			},
			tokenIn:       siusdAddr,
			tokenOut:      usdcAddr,
			amountIn:      "1000000000000000000",
			expectedError: ErrSwapNotSupported,
			expectedGas:   0,
		},
		{
			name: "Unsupported swap (liUSD -> USDC) should fail",
			poolExtra: Extra{
				IsPaused:           false,
				IUSDSupply:         mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets:   mustParseBig("1000000000000000000000000"),
				SIUSDSupply:        mustParseBig("500000000000000000000000"),
				LIUSDSupplies:      []string{"1000000000000000000000000"},
				LIUSDTotalReceipts: []string{"1000000000000000000000000"},
			},
			tokenIn:       liusd1moAddr,
			tokenOut:      usdcAddr,
			amountIn:      "1000000000000000000",
			expectedError: ErrSwapNotSupported,
			expectedGas:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create pool entity with proper token list
			tokens := []*entity.PoolToken{
				{Address: usdcAddr, Decimals: 6, Swappable: true},
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
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	// Actual token addresses from ethereum.json
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
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

	simulator, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Test USDC -> siUSD (mintAndStake)
	amountInUSDC := mustParseBig("1000000")               // 1 USDC
	amountOutSIUSD := mustParseBig("500000000000000000")  // 0.5 siUSD (based on 2:1 rate)
	iusdEquivalent := mustParseBig("1000000000000000000") // 1 iUSD

	simulator.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: amountInUSDC,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  siusdAddr,
			Amount: amountOutSIUSD,
		},
		Fee: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: big.NewInt(0),
		},
	})

	// Check balances updated correctly for mintAndStake
	expectedIUSD := new(big.Int).Add(initialIUSD, iusdEquivalent)
	expectedSIUSDAssets := new(big.Int).Add(initialSIUSDAssets, iusdEquivalent)
	expectedSIUSDSupply := new(big.Int).Add(initialSIUSDSupply, amountOutSIUSD)

	assert.Equal(t, expectedIUSD.String(), simulator.iusdSupply.String())
	assert.Equal(t, expectedSIUSDAssets.String(), simulator.siusdTotalAssets.String())
	assert.Equal(t, expectedSIUSDSupply.String(), simulator.siusdSupply.String())

	// Reset simulator for next test
	simulator, err = NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Test USDC -> liUSD (mintAndLock)
	amountInUSDC2 := mustParseBig("2000000")               // 2 USDC
	amountOutLIUSD := mustParseBig("2000000000000000000")  // 2 liUSD
	iusdEquivalent2 := mustParseBig("2000000000000000000") // 2 iUSD

	simulator.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: amountInUSDC2,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  liusd1moAddr,
			Amount: amountOutLIUSD,
		},
		Fee: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: big.NewInt(0),
		},
	})

	// Check balances updated correctly for mintAndLock
	expectedIUSD2 := new(big.Int).Add(initialIUSD, iusdEquivalent2)
	expectedLIUSD1moSupply := new(big.Int).Add(initialLIUSD1moSupply, amountOutLIUSD)

	assert.Equal(t, expectedIUSD2.String(), simulator.iusdSupply.String())
	assert.Equal(t, expectedLIUSD1moSupply.String(), simulator.liusdSupplies[0].String())
}

func mustParseBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("failed to parse big.Int: " + s)
	}
	return b
}
