package testutil

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
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

					if resIn.RemainingTokenAmountOut != nil && resIn.RemainingTokenAmountOut.Amount.Sign() > 0 {
						amountOut.Sub(amountOut, resIn.RemainingTokenAmountOut.Amount)
						resIn, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
							TokenAmountOut: pool.TokenAmount{
								Token:  tokenOut,
								Amount: amountOut,
							},
							TokenIn: tokenIn,
						})
						require.NoError(tb, err)

						if resIn.RemainingTokenAmountOut != nil && resIn.RemainingTokenAmountOut.Amount.Sign() > 0 {
							amountOut.Sub(amountOut, resIn.RemainingTokenAmountOut.Amount)
							resIn, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
								TokenAmountOut: pool.TokenAmount{
									Token:  tokenOut,
									Amount: amountOut,
								},
								TokenIn: tokenIn,
							})
							require.NoError(tb, err)
						}
						require.True(tb,
							resIn.RemainingTokenAmountOut == nil || resIn.RemainingTokenAmountOut.Amount.Sign() <= 0)

						resOut, err := MustConcurrentSafe(tb, func() (*pool.CalcAmountOutResult, error) {
							return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
								TokenAmountIn: pool.TokenAmount{
									Token:  tokenIn,
									Amount: resIn.TokenAmountIn.Amount,
								},
								TokenOut: tokenOut,
							})
						})
						require.NoError(tb, err)

						finalAmtOut := resOut.TokenAmountOut.Amount
						origAmountOutF, _ := amountOut.Float64()
						finalAmountOutF, _ := finalAmtOut.Float64()
						tb.Logf("amountOut: %s, amountIn: %s, finalAmtOut: %s",
							amountOut, resIn.TokenAmountIn.Amount, finalAmtOut)
						assert.InEpsilonf(tb, origAmountOutF, finalAmountOutF, 1e-4,
							"expected ~%s, got %s", amountOut, finalAmtOut)
					}
				})
			}
		}
	}
}
