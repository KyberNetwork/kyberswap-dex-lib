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

func TestCalcAmountIn(t *testing.T, poolSim interface {
	pool.IPoolSimulator
	pool.IPoolExactOutSimulator
}) {
	for inIdx := range 2 {
		tokenIn, tokenOut := poolSim.GetTokens()[inIdx], poolSim.GetTokens()[1-inIdx]
		baseOut, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  tokenIn,
				Amount: bignumber.BasisPoint,
			},
			TokenOut: tokenOut,
		})
		var base float64
		if err == nil {
			base, _ = baseOut.TokenAmountOut.Amount.Float64()
		}
		for range 32 {
			amountOut, _ := big.NewFloat(max(1, base) * (math.Pow(10, 1+rand.Float64()*9))).Int(nil)
			t.Run(fmt.Sprintf("? token%d -> %s token%d", inIdx, amountOut, 1-inIdx), func(t *testing.T) {
				resIn, err := MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
					return poolSim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{
							Token:  tokenOut,
							Amount: amountOut,
						},
						TokenIn: tokenIn,
					})
				})
				require.NoError(t, err)

				if resIn.RemainingTokenAmountOut != nil && resIn.RemainingTokenAmountOut.Amount.Sign() > 0 {
					amountOut.Sub(amountOut, resIn.RemainingTokenAmountOut.Amount)
					resIn, err = poolSim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{
							Token:  tokenOut,
							Amount: amountOut,
						},
						TokenIn: tokenIn,
					})
					require.NoError(t, err)
				}
				require.True(t,
					resIn.RemainingTokenAmountOut == nil || resIn.RemainingTokenAmountOut.Amount.Sign() == 0)

				resOut, err := MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
					return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{
							Token:  tokenIn,
							Amount: resIn.TokenAmountIn.Amount,
						},
						TokenOut: tokenOut,
					})
				})
				require.NoError(t, err)

				finalAmtOut := resOut.TokenAmountOut.Amount
				origAmountOutF, _ := amountOut.Float64()
				finalAmountOutF, _ := finalAmtOut.Float64()
				t.Logf("amountOut: %s, amountIn: %s, finalAmtOut: %s",
					amountOut, resIn.TokenAmountIn.Amount, finalAmtOut)
				assert.InEpsilonf(t, origAmountOutF, finalAmountOutF, 1e-4,
					"expected ~%s, got %s", amountOut, finalAmtOut)
			})
		}
	}
}
