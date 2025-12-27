package clear

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// createTestPool creates a pool with bidirectional rates
func createTestPool(rate0to1In, rate0to1Out, rate1to0In, rate1to0Out *big.Int) *PoolSimulator {
	extra := Extra{
		Reserves: map[int]map[int]*PreviewSwapResult{
			0: {
				1: {AmountIn: rate0to1In, AmountOut: rate0to1Out},
			},
			1: {
				0: {AmountIn: rate1to0In, AmountOut: rate1to0Out},
			},
		},
	}
	extraBytes, _ := json.Marshal(extra)

	staticExtra := StaticExtra{
		SwapAddress: "0xswap",
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	entityPool := entity.Pool{
		Address:  "0xvault",
		Exchange: "clear",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xUSDC", Decimals: 6, Swappable: true},
			{Address: "0xGHO", Decimals: 18, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, _ := NewPoolSimulator(entityPool)
	return sim
}

func TestPoolSimulator_CalcAmountOut_Bidirectional(t *testing.T) {
	t.Parallel()

	// Create pool with bidirectional rates:
	// 0→1: 1 USDC (1e6) gives 1 GHO (1e18) - rate 1:1
	// 1→0: 1 GHO (1e18) gives 0.95 USDC (0.95e6) - rate 1:0.95 (simulating depeg)
	sim := createTestPool(
		big.NewInt(1000000),           // 0→1: 1e6 USDC in
		big.NewInt(1000000000000000000), // 0→1: 1e18 GHO out
		big.NewInt(1000000000000000000), // 1→0: 1e18 GHO in
		big.NewInt(950000),            // 1→0: 0.95e6 USDC out
	)

	tests := []struct {
		name        string
		tokenIn     string
		tokenOut    string
		amountIn    *big.Int
		expectedOut *big.Int
	}{
		{
			name:        "USDC to GHO (1:1 rate)",
			tokenIn:     "0xUSDC",
			tokenOut:    "0xGHO",
			amountIn:    big.NewInt(1000000), // 1 USDC
			expectedOut: big.NewInt(1000000000000000000), // 1 GHO
		},
		{
			name:        "GHO to USDC (depeg rate 1:0.95)",
			tokenIn:     "0xGHO",
			tokenOut:    "0xUSDC",
			amountIn:    big.NewInt(1000000000000000000), // 1 GHO
			expectedOut: big.NewInt(950000), // 0.95 USDC
		},
		{
			name:        "USDC to GHO - larger amount",
			tokenIn:     "0xUSDC",
			tokenOut:    "0xGHO",
			amountIn:    big.NewInt(100000000), // 100 USDC
			expectedOut: new(big.Int).Mul(big.NewInt(100), big.NewInt(1000000000000000000)), // 100 GHO
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: tc.amountIn,
				},
				TokenOut: tc.tokenOut,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedOut.String(), result.TokenAmountOut.Amount.String())
		})
	}
}

func TestPoolSimulator_CalcAmountOut_NoRate(t *testing.T) {
	t.Parallel()

	// Pool with no rates (Clear refuses swaps - no depeg)
	sim := createTestPool(nil, nil, nil, nil)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xUSDC",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xGHO",
	})

	// Should return 0, not error
	require.NoError(t, err)
	assert.Equal(t, "0", result.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_ZeroRate(t *testing.T) {
	t.Parallel()

	// Pool with zero AmountOut (Clear returned 0)
	sim := createTestPool(
		big.NewInt(1000000),
		big.NewInt(0), // Zero output
		big.NewInt(1000000000000000000),
		big.NewInt(0),
	)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xUSDC",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xGHO",
	})

	require.NoError(t, err)
	assert.Equal(t, "0", result.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()

	sim := createTestPool(
		big.NewInt(1000000),
		big.NewInt(1000000000000000000),
		big.NewInt(1000000000000000000),
		big.NewInt(950000),
	)

	cloned := sim.CloneState()
	require.NotNil(t, cloned)

	clonedSim, ok := cloned.(*PoolSimulator)
	require.True(t, ok)

	// Verify it's a different instance
	assert.NotSame(t, sim, clonedSim)
	assert.NotSame(t, sim.RWMutex, clonedSim.RWMutex)

	// Verify reserves are cloned (not shared)
	assert.NotSame(t, sim.extra.Reserves[0][1], clonedSim.extra.Reserves[0][1])

	// Verify values are equal
	assert.Equal(t,
		sim.extra.Reserves[0][1].AmountOut.String(),
		clonedSim.extra.Reserves[0][1].AmountOut.String(),
	)
}

func TestPoolSimulator_InvalidToken(t *testing.T) {
	t.Parallel()

	sim := createTestPool(
		big.NewInt(1000000),
		big.NewInt(1000000000000000000),
		big.NewInt(1000000000000000000),
		big.NewInt(950000),
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xINVALID",
			Amount: big.NewInt(1000000),
		},
		TokenOut: "0xGHO",
	})

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestPoolSimulator_InvalidAmountIn(t *testing.T) {
	t.Parallel()

	sim := createTestPool(
		big.NewInt(1000000),
		big.NewInt(1000000000000000000),
		big.NewInt(1000000000000000000),
		big.NewInt(950000),
	)

	tests := []struct {
		name     string
		amountIn *big.Int
	}{
		{name: "nil amount", amountIn: nil},
		{name: "zero amount", amountIn: big.NewInt(0)},
		{name: "negative amount", amountIn: big.NewInt(-100)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xUSDC",
					Amount: tc.amountIn,
				},
				TokenOut: "0xGHO",
			})

			assert.ErrorIs(t, err, ErrInvalidAmountIn)
		})
	}
}
