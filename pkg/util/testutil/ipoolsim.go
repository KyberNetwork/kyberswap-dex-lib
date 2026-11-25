package testutil

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtOutIPoolSim tests native uint256 CalcAmtOut with given input and output amounts, like TestCalcAmountOut.
// Empty output means error expected.
func TestCalcAmtOutIPoolSim[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSim, expected map[int]map[int]map[string]string) {
	tb.Helper()
	tokens := poolSim.GetTokens()
	for idxIn, expected := range expected {
		for idxOut, expected := range expected {
			for amtIn, expected := range expected {
				testIPoolSim(tb, poolSim, amtIn, expected, idxIn, idxOut, tokens)
			}
		}
	}
}

func testIPoolSim[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSim, amtIn string, expected string, idxIn, idxOut int, tokens []string) {
	tb.Helper()
	tb.Run(fmt.Sprintf("%s token%d -> ? token%d", amtIn, idxIn, idxOut), func(tb TB) {
		tb.Helper()
		var in uint256.Int
		require.NoError(tb, in.SetFromDecimal(amtIn))
		res, err := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokens[idxIn], Amount: in},
			TokenOut:      tokens[idxOut],
		})
		if expected == "" {
			require.Error(tb, err)
			return
		}
		require.NoError(tb, err)
		var want uint256.Int
		require.NoError(tb, want.SetFromDecimal(expected))
		assert.Equal(tb, want.String(), res.TokenAmountOut.Amount.String())
	})
}

// TestCalcAmountInIPoolSim tests native uint256 CalcAmtIn with given output and input amounts, like TestCalcAmtOutIPoolSim
// but for exact-out. The map is keyed by tokenIn index, then tokenOut index, then the output amount; the value is the
// expected input amount. Empty input means error expected.
func TestCalcAmountInIPoolSim[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSim, expected map[int]map[int]map[string]string) {
	tb.Helper()
	tokens := poolSim.GetTokens()
	for idxIn, expected := range expected {
		for idxOut, expected := range expected {
			for amtOut, expected := range expected {
				testIPoolSimCalcAmtIn(tb, poolSim, amtOut, expected, idxIn, idxOut, tokens)
			}
		}
	}
}

func testIPoolSimCalcAmtIn[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim pool.IPoolSim, amtOut string, expected string, idxIn, idxOut int, tokens []string) {
	tb.Helper()
	tb.Run(fmt.Sprintf("%s token%d <- ? token%d", amtOut, idxIn, idxOut), func(tb TB) {
		tb.Helper()
		var out uint256.Int
		require.NoError(tb, out.SetFromDecimal(amtOut))
		calc, ok := poolSim.(pool.ICalcAmtIn)
		require.True(tb, ok, "pool does not implement native CalcAmtIn")
		res, err := calc.CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: tokens[idxOut], Amount: out},
			TokenIn:        tokens[idxIn],
		})
		if expected == "" {
			require.Error(tb, err)
			return
		}
		require.NoError(tb, err)
		var want uint256.Int
		require.NoError(tb, want.SetFromDecimal(expected))
		assert.Equal(tb, want.String(), res.TokenAmountIn.Amount.String())
	})
}

// TestCloneStateIPoolSim checks that UpdateState on a clone leaves the original's quote unchanged.
func TestCloneStateIPoolSim(t testing.TB, sim pool.IPoolSim, params pool.CalcAmtOutParams) {
	t.Helper()

	clone := sim.CloneState()

	result1, err := sim.CalcAmtOut(params)
	require.NoError(t, err)

	clone.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *result1.TokenAmountOut,
		SwapInfo:       result1.SwapInfo,
		SwapLimit:      params.Limit,
	})

	result2, err := sim.CalcAmtOut(params)
	require.NoError(t, err)

	assert.Equal(t, result1.TokenAmountOut.Amount.String(), result2.TokenAmountOut.Amount.String(),
		"CloneState: UpdateState on clone mutated the original")
}
