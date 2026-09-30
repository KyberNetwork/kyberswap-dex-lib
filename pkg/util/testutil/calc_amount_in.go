package testutil

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// TestCalcAmountIn tests CalcAmountIn with generated sensible inputs
func TestCalcAmountIn[TB interface {
	testing.TB
	Run(string, func(TB)) bool
}](tb TB, poolSim interface {
	pool.IPoolSimulator
	pool.IPoolExactOutSimulator
}, runsOpt ...int) {
	tb.Helper()
	runs := 32
	if len(runsOpt) > 0 {
		runs = runsOpt[0]
	}
	tokens := poolSim.GetTokens()
	for idxIn, tokenIn := range tokens {
		tokenOuts := poolSim.CanSwapFrom(tokenIn)
		for _, tokenOut := range tokenOuts {
			idxOut := poolSim.GetTokenIndex(tokenOut)
			var base float64
			for _, exp := range []int{3, 4, 6, 9, 13} {
				baseOut, err := pool.CalcAmountOut(
					ctx,
					poolSim,
					pool.TokenAmount{Token: tokenIn, Amount: bignumber.TenPowInt(exp)},
					tokenOut,
					nil,
				)
				if err == nil {
					base, _ = baseOut.TokenAmountOut.Amount.Float64()
					break
				}
			}
			base = max(1, base)
			maxExp := 1.0
			for _, exp := range []int{23, 17, 12, 8, 5} {
				if baseOut, err := pool.CalcAmountOut(
					ctx,
					poolSim,
					pool.TokenAmount{Token: tokenIn, Amount: bignumber.TenPowInt(exp)},
					tokenOut,
					nil,
				); err == nil {
					maxExp, _ = baseOut.TokenAmountOut.Amount.Float64()
					maxExp = math.Log10(maxExp/base) - 1
					break
				}
			}
			for range runs {
				amountOut, _ := big.NewFloat(base * (math.Pow(10, 1+rand.Float64()*maxExp))).Int(nil)
				tb.Run(fmt.Sprintf("? token%d -> %s token%d", idxIn, amountOut, idxOut), func(tb TB) {
					tb.Helper()
					// asking again for only the filled part must eventually leave no remaining amount out
					for range 3 {
						resIn, err := MustConcurrentSafe(tb, func() (*pool.CalcAmountInResult, error) {
							return poolSim.CalcAmountIn(pool.CalcAmountInParams{
								TokenAmountOut: pool.TokenAmount{
									Token:  tokenOut,
									Amount: amountOut,
								},
								TokenIn: tokenIn,
							})
						})
						require.NoError(tb, err)

						// amountIn only buys amountOut - remaining
						filledOut := amountOut
						if resIn.RemainingTokenAmountOut != nil && resIn.RemainingTokenAmountOut.Amount.Sign() > 0 {
							require.Negative(tb, resIn.RemainingTokenAmountOut.Amount.Cmp(amountOut),
								"remaining %s >= amountOut %s", resIn.RemainingTokenAmountOut.Amount, amountOut)
							filledOut = new(big.Int).Sub(amountOut, resIn.RemainingTokenAmountOut.Amount)
						}
						assertRoundTrip(tb, poolSim, tokenIn, tokenOut, resIn.TokenAmountIn.Amount, filledOut)
						if filledOut == amountOut {
							return
						}
						amountOut = filledOut
					}
					tb.Errorf("still has remaining amount out when asking for %s", amountOut)
				})
			}
		}
	}
}

// roundTripAbsTolerance absorbs rounding when amountOut is too small for roundTripRelTolerance.
const roundTripRelTolerance, roundTripAbsTolerance = 2e-4, 3

// assertRoundTrip checks CalcAmountOut(amountIn) is close to wantOut. A larger overshoot is fine when amountIn is
// minimal (1 wei less falls short), e.g. an order book filling whole lots.
func assertRoundTrip(tb testing.TB, poolSim pool.IPoolSimulator, tokenIn, tokenOut string, amountIn, wantOut *big.Int) {
	tb.Helper()
	calcAmountOut := func(amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
		return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  tokenIn,
				Amount: amountIn,
			},
			TokenOut: tokenOut,
		})
	}
	resOut, err := MustConcurrentSafe(tb, func() (*pool.CalcAmountOutResult, error) { return calcAmountOut(amountIn) })
	require.NoError(tb, err)

	gotOut := resOut.TokenAmountOut.Amount
	tb.Logf("amountOut: %s, amountIn: %s, finalAmtOut: %s", wantOut, amountIn, gotOut)
	var diff big.Int
	diff.Sub(gotOut, wantOut).Abs(&diff)
	diffF, _ := diff.Float64()
	wantOutF, _ := wantOut.Float64()
	if diff.IsInt64() && diff.Int64() <= roundTripAbsTolerance || diffF <= roundTripRelTolerance*wantOutF {
		return
	} else if gotOut.Cmp(wantOut) > 0 {
		resLess, err := calcAmountOut(new(big.Int).Sub(amountIn, bignumber.One))
		if err != nil || resLess.TokenAmountOut.Amount.Cmp(wantOut) < 0 {
			return
		}
	}
	tb.Errorf("expected ~%s, got %s", wantOut, gotOut)
}
