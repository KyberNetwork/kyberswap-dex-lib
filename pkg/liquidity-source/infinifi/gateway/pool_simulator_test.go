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
	// Example token addresses (replace with actual addresses)
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	iusdAddr := "0x1234567890123456789012345678901234567890"  // Replace with actual
	siusdAddr := "0x2234567890123456789012345678901234567890" // Replace with actual
	liusdAddr := "0x3234567890123456789012345678901234567890" // Replace with actual

	testCases := []struct {
		name           string
		poolExtra      Extra
		tokenIn        string
		tokenOut       string
		amountIn       string
		expectedAmount string
		expectedError  error
	}{
		{
			name: "USDC -> iUSD (Mint) - 1:1 with decimal conversion",
			poolExtra: Extra{
				IsPaused:         false,
				IUSDSupply:       mustParseBig("1000000000000000000000000"), // 1M iUSD (18 decimals)
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"),  // 500k iUSD
				SIUSDSupply:      mustParseBig("500000000000000000000000"),  // 500k siUSD
				LIUSDSupplies:    []string{"100000000000000000000000"},      // 100k liUSD
			},
			tokenIn:        usdcAddr,
			tokenOut:       iusdAddr,
			amountIn:       "1000000",             // 1 USDC (6 decimals)
			expectedAmount: "1000000000000000000", // 1 iUSD (18 decimals)
			expectedError:  nil,
		},
		{
			name: "iUSD -> siUSD (Stake) - ERC4626 conversion",
			poolExtra: Extra{
				IsPaused:         false,
				IUSDSupply:       mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"), // 500k iUSD in vault
				SIUSDSupply:      mustParseBig("400000000000000000000000"), // 400k siUSD shares
				LIUSDSupplies:    []string{"100000000000000000000000"},
			},
			tokenIn:  iusdAddr,
			tokenOut: siusdAddr,
			amountIn: "1000000000000000000", // 1 iUSD
			// Expected: 1 iUSD * 400k siUSD / 500k iUSD = 0.8 siUSD
			expectedAmount: "800000000000000000",
			expectedError:  nil,
		},
		{
			name: "iUSD -> liUSD (Lock) - 1:1",
			poolExtra: Extra{
				IsPaused:         false,
				IUSDSupply:       mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"),
				SIUSDSupply:      mustParseBig("500000000000000000000000"),
				LIUSDSupplies:    []string{"100000000000000000000000"},
			},
			tokenIn:        iusdAddr,
			tokenOut:       liusdAddr,
			amountIn:       "1000000000000000000", // 1 iUSD
			expectedAmount: "1000000000000000000", // 1 liUSD
			expectedError:  nil,
		},
		{
			name: "Contract paused should fail",
			poolExtra: Extra{
				IsPaused:         true,
				IUSDSupply:       mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"),
				SIUSDSupply:      mustParseBig("500000000000000000000000"),
				LIUSDSupplies:    []string{},
			},
			tokenIn:       usdcAddr,
			tokenOut:      iusdAddr,
			amountIn:      "1000000",
			expectedError: ErrContractPaused,
		},
		{
			name: "Async redemption (iUSD -> USDC) should fail",
			poolExtra: Extra{
				IsPaused:         false,
				IUSDSupply:       mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"),
				SIUSDSupply:      mustParseBig("500000000000000000000000"),
				LIUSDSupplies:    []string{},
			},
			tokenIn:       iusdAddr,
			tokenOut:      usdcAddr,
			amountIn:      "1000000000000000000",
			expectedError: ErrAsyncRedemption,
		},
		{
			name: "Async unstake (siUSD -> iUSD) should fail",
			poolExtra: Extra{
				IsPaused:         false,
				IUSDSupply:       mustParseBig("1000000000000000000000000"),
				SIUSDTotalAssets: mustParseBig("500000000000000000000000"),
				SIUSDSupply:      mustParseBig("500000000000000000000000"),
				LIUSDSupplies:    []string{},
			},
			tokenIn:       siusdAddr,
			tokenOut:      iusdAddr,
			amountIn:      "1000000000000000000",
			expectedError: ErrAsyncRedemption,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create pool entity
			poolEntity := entity.Pool{
				Address:  "0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5",
				Exchange: "infinifi",
				Type:     DexType,
				Tokens: []*entity.PoolToken{
					{Address: usdcAddr, Decimals: 6, Swappable: true},
					{Address: iusdAddr, Decimals: 18, Swappable: true},
					{Address: siusdAddr, Decimals: 18, Swappable: true},
					{Address: liusdAddr, Decimals: 18, Swappable: true},
				},
				Reserves: []string{
					tc.poolExtra.IUSDSupply.String(),
					tc.poolExtra.SIUSDTotalAssets.String(),
					tc.poolExtra.SIUSDSupply.String(),
					"100000000000000000000000",
				},
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
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedAmount, result.TokenAmountOut.Amount.String())
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	usdcAddr := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	iusdAddr := "0x1234567890123456789012345678901234567890"
	siusdAddr := "0x2234567890123456789012345678901234567890"

	poolExtra := Extra{
		IsPaused:         false,
		IUSDSupply:       mustParseBig("1000000000000000000000000"),
		SIUSDTotalAssets: mustParseBig("500000000000000000000000"),
		SIUSDSupply:      mustParseBig("500000000000000000000000"),
		LIUSDSupplies:    []string{},
	}

	poolEntity := entity.Pool{
		Address:  "0x3f04b65ddbd87f9ce0a2e7eb24d80e7fb87625b5",
		Exchange: "infinifi",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Decimals: 6, Swappable: true},
			{Address: iusdAddr, Decimals: 18, Swappable: true},
			{Address: siusdAddr, Decimals: 18, Swappable: true},
		},
		Reserves: []string{
			poolExtra.IUSDSupply.String(),
			poolExtra.SIUSDTotalAssets.String(),
			poolExtra.SIUSDSupply.String(),
		},
	}

	extraBytes, err := json.Marshal(poolExtra)
	require.NoError(t, err)
	poolEntity.Extra = string(extraBytes)

	simulator, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Test USDC -> iUSD (mint)
	amountIn := mustParseBig("1000000")              // 1 USDC
	amountOut := mustParseBig("1000000000000000000") // 1 iUSD

	initialIUSD := new(big.Int).Set(simulator.iusdSupply)

	simulator.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: amountIn,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  iusdAddr,
			Amount: amountOut,
		},
		Fee: pool.TokenAmount{
			Token:  usdcAddr,
			Amount: big.NewInt(0),
		},
	})

	// Check balances updated correctly
	expectedIUSD := new(big.Int).Add(initialIUSD, amountOut)
	assert.Equal(t, expectedIUSD.String(), simulator.iusdSupply.String())
}

func mustParseBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("failed to parse big.Int: " + s)
	}
	return b
}
