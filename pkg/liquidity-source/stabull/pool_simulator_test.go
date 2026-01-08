package stabull

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPoolSimulator tests pool simulator creation
func TestNewPoolSimulator(t *testing.T) {
	extra := Extra{
		CurveParams: CurveParameters{
			Alpha:   "1000000000000000000",
			Beta:    "500000000000000000",
			Delta:   "100000000000000000",
			Epsilon: "200000000000000000",
			Lambda:  "1000000000000000000",
		},
		OracleRate: "1000000000000000000",
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xtest",
		Exchange: "stabull",
		Type:     "stabull",
		Reserves: []string{"1000000000000000000000", "2000000000000000000000"}, // 1000, 2000 tokens
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0"},
			{Address: "0xtoken1"},
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, 2, len(sim.Info.Tokens))
	assert.Equal(t, 2, len(sim.Info.Reserves))
}

// TestCalcAmountOut tests the swap calculation
func TestCalcAmountOut(t *testing.T) {
	tests := []struct {
		name        string
		reserveIn   string
		reserveOut  string
		amountIn    string
		oracleRate  string
		swapFee     string
		expectedOut string // Expected output calculated using Stabull curve formula
		expectError bool
	}{
		{
			name:       "Basic swap - 1 token in",
			reserveIn:  "1000000000000000000000", // 1000 tokens
			reserveOut: "2000000000000000000000", // 2000 tokens
			amountIn:   "1000000000000000000",    // 1 token
			oracleRate: "1000000000000000000",    // 1.0
			swapFee:    "30",                     // 0.3%
			// Expected output calculated using Stabull curve formula with greek parameters
			// Formula incorporates alpha, beta, delta, epsilon, lambda for dynamic pricing
			expectedOut: "1598721023181454836", // ~1.598 tokens out for 1 token in
			expectError: false,
		},
		{
			name:       "Large swap",
			reserveIn:  "1000000000000000000000",
			reserveOut: "2000000000000000000000",
			amountIn:   "100000000000000000000", // 100 tokens
			oracleRate: "1000000000000000000",
			swapFee:    "30",
			// Large swap has more slippage due to curve parameters (beta, delta)
			expectedOut: "181487543189670849245", // ~181.48 tokens out for 100 tokens in
			expectError: false,
		},
		{
			name:        "Zero amount in",
			reserveIn:   "1000000000000000000000",
			reserveOut:  "2000000000000000000000",
			amountIn:    "0",
			oracleRate:  "1000000000000000000",
			swapFee:     "30",
			expectedOut: "0",
			expectError: true,
		},
		{
			name:       "Large amount (approaches reserve limit)",
			reserveIn:  "1000000000000000000000",
			reserveOut: "2000000000000000000000",
			amountIn:   "999999000000000000000000", // Huge amount
			oracleRate: "1000000000000000000",
			swapFee:    "30",
			// Stabull curve with hybrid invariant approaches but never exceeds reserveOut
			// The formula prevents draining the pool completely
			expectedOut: "1998001995606787840505", // ~1998 tokens out, leaving reserves
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pool simulator with proper Extra JSON
			extra := Extra{
				CurveParams: CurveParameters{
					Alpha:   "1000000000000000000",
					Beta:    "500000000000000000",
					Delta:   "100000000000000000",
					Epsilon: "200000000000000000",
					Lambda:  "1000000000000000000",
				},
				OracleRate: tt.oracleRate,
			}
			extraBytes, _ := json.Marshal(extra)

			entityPool := entity.Pool{
				Address:  "0xtest",
				Exchange: "stabull",
				Type:     "stabull",
				Reserves: []string{tt.reserveIn, tt.reserveOut},
				Tokens: []*entity.PoolToken{
					{Address: "0xtoken0"},
					{Address: "0xtoken1"},
				},
				Extra: string(extraBytes),
			}

			sim, err := NewPoolSimulator(entityPool)
			require.NoError(t, err)

			// Parse amount as decimal string
			amountIn, ok := new(big.Int).SetString(tt.amountIn, 10)
			require.True(t, ok, "Failed to parse amountIn: %s", tt.amountIn)

			// Calculate amount out
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xtoken0",
					Amount: amountIn,
				},
				TokenOut: "0xtoken1",
			})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Validate expected output matches the Stabull curve calculation
				expected, ok := new(big.Int).SetString(tt.expectedOut, 10)
				require.True(t, ok, "Failed to parse expectedOut: %s", tt.expectedOut)
				assert.Equal(t, expected, result.TokenAmountOut.Amount,
					"Output mismatch: expected %s, got %s", expected, result.TokenAmountOut.Amount)

				// Additional sanity checks
				assert.NotNil(t, result.TokenAmountOut)
				assert.True(t, result.TokenAmountOut.Amount.Cmp(big.NewInt(0)) > 0,
					"Output should be positive")
				assert.True(t, result.Gas > 0, "Gas should be positive")
			}
		})
	}
}

// TestUpdateBalance tests state updates after swaps
func TestUpdateBalance(t *testing.T) {
	// Use bignumber.NewBig10 or string parsing for large values
	initialReserve0, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000 tokens
	initialReserve1, _ := new(big.Int).SetString("2000000000000000000000", 10) // 2000 tokens
	amountIn, _ := new(big.Int).SetString("1000000000000000000", 10)           // 1 token
	amountOut, _ := new(big.Int).SetString("1990000000000000000", 10)          // ~1.99 tokens (example)

	extra := Extra{
		CurveParams: CurveParameters{
			Alpha:   "1000000000000000000",
			Beta:    "500000000000000000",
			Delta:   "100000000000000000",
			Epsilon: "200000000000000000",
			Lambda:  "1000000000000000000",
		},
		OracleRate: "1000000000000000000",
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xtest",
		Exchange: "stabull",
		Type:     "stabull",
		Reserves: []string{initialReserve0.String(), initialReserve1.String()},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0"},
			{Address: "0xtoken1"},
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	// Perform swap
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtoken0",
			Amount: amountIn,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  "0xtoken1",
			Amount: amountOut,
		},
	})

	// Check reserves updated
	assert.True(t, sim.Info.Reserves[0].Cmp(initialReserve0) > 0,
		"Reserve0 should increase")
	assert.True(t, sim.Info.Reserves[1].Cmp(initialReserve1) < 0,
		"Reserve1 should decrease")

	// Fee stays in pool (goes to LPs), so reserve increases by full amountIn
	actualIncrease := new(big.Int).Sub(sim.Info.Reserves[0], initialReserve0)
	assert.Equal(t, amountIn, actualIncrease,
		"Reserve0 increase should match full input amount (fee stays in pool)")
}

// TestCanSwap tests token swap compatibility
func TestCanSwap(t *testing.T) {
	extra := Extra{
		CurveParams: CurveParameters{
			Alpha:   "1000000000000000000",
			Beta:    "500000000000000000",
			Delta:   "100000000000000000",
			Epsilon: "200000000000000000",
			Lambda:  "1000000000000000000",
		},
		OracleRate: "1000000000000000000",
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xtest",
		Exchange: "stabull",
		Type:     "stabull",
		Reserves: []string{"1000000000000000000000", "2000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0"},
			{Address: "0xtoken1"},
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	// Can swap token0 to token1
	result := sim.CanSwapTo("0xtoken0")
	assert.NotNil(t, result)
	assert.Contains(t, result, "0xtoken1")

	// Can swap token1 to token0
	result = sim.CanSwapFrom("0xtoken1")
	assert.NotNil(t, result)
	assert.Contains(t, result, "0xtoken0")

	// Cannot swap unknown token
	result = sim.CanSwapTo("0xunknown")
	assert.Nil(t, result)
}

// TestGetMetaInfo tests metadata retrieval
func TestGetMetaInfo(t *testing.T) {
	curveParams := CurveParameters{
		Alpha:   "1000000000000000000",
		Beta:    "500000000000000000",
		Delta:   "100000000000000000",
		Epsilon: "200000000000000000",
		Lambda:  "1000000000000000000",
	}
	extra := Extra{
		CurveParams: curveParams,
		OracleRate:  "1500000000000000000",
	}
	extraBytes, _ := json.Marshal(extra)

	entityPool := entity.Pool{
		Address:  "0xtest",
		Exchange: "stabull",
		Type:     "stabull",
		Reserves: []string{"1000000000000000000000", "2000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0"},
			{Address: "0xtoken1"},
		},
		Extra: string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	meta := sim.GetMetaInfo("", "")
	require.NotNil(t, meta)

	metaTyped, ok := meta.(Meta)
	require.True(t, ok, "Meta should be of type Meta")
	assert.Equal(t, "1000000000000000000", metaTyped.Alpha)
	assert.Equal(t, "1500000000000000000", metaTyped.OracleRate)
}

// BenchmarkCalcAmountOut benchmarks swap calculation performance
func BenchmarkCalcAmountOut(b *testing.B) {
	entityPool := entity.Pool{
		Address:  "0xtest",
		Exchange: "stabull",
		Type:     "stabull",
		Reserves: []string{"1000000000000000000000", "2000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0"},
			{Address: "0xtoken1"},
		},
		Extra: `{"oracleRate":"1000000000000000000","swapFee":"30"}`,
	}

	sim, _ := NewPoolSimulator(entityPool)
	amountIn := big.NewInt(1000000000000000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xtoken0",
				Amount: amountIn,
			},
			TokenOut: "0xtoken1",
		})
	}
}
