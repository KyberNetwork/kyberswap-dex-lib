package machima

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCalcAmtInLegacyParity runs native CalcAmtIn and the legacy big.Int CalcAmountIn over the same pools, both
// directions and amounts up to 2^256-1, in sequence: after each success both sims apply UpdateState with their
// own SwapInfo, so the next quote also checks state. Amount, gas, SwapInfo and errors must match to the wei.
func TestCalcAmtInLegacyParity(t *testing.T) {
	maxU := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	amounts := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(10), big.NewInt(1e9), big.NewInt(1e14),
		big.NewInt(1e17), new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
		new(big.Int).Lsh(big.NewInt(1), 128), maxU}
	dirs := [][2]string{{wethAddr, tokenAddr}, {tokenAddr, wethAddr}}
	successes := 0
	for name, build := range calcAmtInVariants {
		for _, dir := range dirs {
			native, legacy := build(t), build(t)
			for step := range 2 {
				for _, amt := range amounts {
					label := fmt.Sprintf("%s %s->%s amountOut=%s step=%d", name, dir[0][len(dir[0])-4:],
						dir[1][len(dir[1])-4:], amt, step)
					lRes, lErr := legacy.CalcAmountIn(pool.CalcAmountInParams{TokenIn: dir[0],
						TokenAmountOut: pool.TokenAmount{Token: dir[1], Amount: new(big.Int).Set(amt)}})
					nRes, nErr := native.CalcAmtIn(pool.CalcAmtInParams{TokenIn: dir[0],
						TokenAmountOut: pool.TokenAmt{Token: dir[1], Amount: *uint256.MustFromBig(amt)}})
					if lErr != nil || nErr != nil {
						require.Equal(t, fmt.Sprint(lErr), fmt.Sprint(nErr), label+" error")
						continue
					}
					successes++
					require.Equal(t, lRes.TokenAmountIn.Amount.String(), nRes.TokenAmountIn.Amount.Dec(), label+" in")
					require.Equal(t, lRes.Gas, nRes.Gas, label+" gas")
					require.Equal(t, lRes.SwapInfo, nRes.SwapInfo, label+" swapinfo")
					// Advance both by exact-in of the quoted amount.
					lOut, err := legacy.CalcAmountOut(pool.CalcAmountOutParams{TokenOut: dir[1],
						TokenAmountIn: pool.TokenAmount{Token: dir[0], Amount: lRes.TokenAmountIn.Amount}})
					if err != nil {
						continue
					}
					nOut, err := native.CalcAmtOut(pool.CalcAmtOutParams{TokenOut: dir[1],
						TokenAmountIn: pool.TokenAmt{Token: dir[0], Amount: nRes.TokenAmountIn.Amount}})
					require.NoError(t, err, label)
					legacy.UpdateState(pool.UpdateStateParams{SwapInfo: lOut.SwapInfo,
						TokenAmountIn: pool.TokenAmt{Token: dir[0]}, TokenAmountOut: pool.TokenAmt{Token: dir[1]}})
					native.UpdateState(pool.UpdateStateParams{SwapInfo: nOut.SwapInfo,
						TokenAmountIn: pool.TokenAmt{Token: dir[0]}, TokenAmountOut: pool.TokenAmt{Token: dir[1]}})
				}
			}
		}
	}
	require.Positive(t, successes, "no quote succeeded; the parity check proves nothing")
}
