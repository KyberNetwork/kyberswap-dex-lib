package uniswapv3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// rangePool is a 0.3% pool at price 1, tick spacing 1, with liquidity 1e18 between ticks -600 and
// 600 and 1000 over the full range, and reserves of 1e18 of each token, far more than the ticks can
// pay out: about 2.96e16 of either token before the price leaves the range, then next to nothing,
// as balances that out-of-range positions and fees leave behind are.
func rangePool(t *testing.T) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x0000000000000000000000000000000000000001",
		Exchange: "uniswapv3",
		Type:     DexTypeUniswapV3,
		SwapFee:  3000,
		Reserves: entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens:   []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		Extra: `{"liquidity":1000000000000001000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":1,` +
			`"tick":0,"ticks":[{"index":-887272,"liquidityGross":1000,"liquidityNet":1000},` +
			`{"index":-600,"liquidityGross":1000000000000000000,"liquidityNet":1000000000000000000},` +
			`{"index":600,"liquidityGross":1000000000000000000,"liquidityNet":-1000000000000000000},` +
			`{"index":887272,"liquidityGross":1000,"liquidityNet":-1000}]}`,
	}, 0)
	require.NoError(t, err)
	return sim
}

// TestCalcAmountInBeyondLiquidity: an output the ticks cannot pay is paid only in part, though the
// pool holds that much of the token: the pool's own swap reaches the price limit and pays less.
// CalcAmountIn reports the part it does not pay, which the exact-in swap of its input confirms.
func TestCalcAmountInBeyondLiquidity(t *testing.T) {
	sim := rangePool(t)
	for i, in := range sim.Info.Tokens {
		out, amountOut := sim.Info.Tokens[1-i], big.NewInt(1e17)
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: out, Amount: amountOut}, TokenIn: in})
		require.NoError(t, err)
		require.NotNil(t, res.RemainingTokenAmountOut, "%s in for 1e17 %s out", in, out)
		owed := res.RemainingTokenAmountOut.Amount
		require.Positive(t, owed.Sign(), "%s in for 1e17 %s out: the unpaid output", in, out)
		paid := new(big.Int).Sub(amountOut, owed)

		got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *res.TokenAmountIn, TokenOut: out})
		require.NoError(t, err)
		assert.LessOrEqual(t, new(big.Int).Sub(got.TokenAmountOut.Amount, paid).CmpAbs(big.NewInt(1)), 0,
			"%s in: exact in of the quoted input pays %s, the quote says %s", in, got.TokenAmountOut.Amount, paid)

		res, err = sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: out, Amount: big.NewInt(1e16)}, TokenIn: in})
		require.NoError(t, err)
		assert.Zero(t, res.RemainingTokenAmountOut.Amount.Sign(), "an output the range can pay is paid whole")
	}
}

// TestCalcAmountInNextStateIsExactIn: a swap is always executed exact in, so the state a quote by
// output hands UpdateBalance is the state the exact-in swap of its input leaves, which a quote by
// that input reports, not the end of the exact-out walk, whose rounding stops at another price.
func TestCalcAmountInNextStateIsExactIn(t *testing.T) {
	sim := rangePool(t)
	for i, in := range sim.Info.Tokens {
		out := sim.Info.Tokens[1-i]
		for _, amountOut := range []int64{1, 999, 123456789, 7e12, 3e15, 2.9e16} {
			ri, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: big.NewInt(amountOut)}, TokenIn: in})
			require.NoError(t, err)
			ro, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *ri.TokenAmountIn, TokenOut: out})
			require.NoError(t, err)
			byOut, byIn := ri.SwapInfo.(SwapInfo), ro.SwapInfo.(SwapInfo)
			assert.Equal(t, byIn.NextStateSqrtRatioX96.Dec(), byOut.NextStateSqrtRatioX96.Dec(),
				"%s in for %d out: price", in, amountOut)
			assert.Equal(t, byIn.NextStateLiquidity.Dec(), byOut.NextStateLiquidity.Dec(),
				"%s in for %d out: liquidity", in, amountOut)
			assert.Equal(t, byIn.NextStateTickCurrent, byOut.NextStateTickCurrent, "%s in for %d out: tick", in, amountOut)
		}
	}
}
