package integral

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// quoteGrid quotes every amount 10^0..10^30 both ways through both fixture pools, exact-in and
// exact-out, and renders each result or error.
func quoteGrid(t testing.TB) []string {
	var lines []string
	tokenAmount := func(ta *pool.TokenAmount) string {
		if ta == nil || ta.Amount == nil {
			return "-"
		}
		return ta.Amount.String()
	}
	for _, sim := range []*PoolSimulator{ps, thenaPS} {
		tokens := sim.GetTokens()
		for dir := range 2 {
			tokenIn, tokenOut := tokens[dir], tokens[1-dir]
			for k := range 31 {
				amount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
				out, err := sim.CloneState().CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amount}, TokenOut: tokenOut})
				if err != nil {
					lines = append(lines, fmt.Sprintf("out %s %d %d: %v", sim.GetAddress(), dir, k, err))
				} else {
					su := out.SwapInfo.(StateUpdate)
					lines = append(lines, fmt.Sprintf("out %s %d %d: %s %s %s %d %s %s %d", sim.GetAddress(), dir, k,
						tokenAmount(out.TokenAmountOut), tokenAmount(out.Fee), tokenAmount(out.RemainingTokenAmountIn),
						out.Gas, su.Liquidity.Dec(), su.Price.Dec(), su.Tick))
				}
				in, err := sim.CloneState().(*PoolSimulator).CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: amount}, TokenIn: tokenIn})
				if err != nil {
					lines = append(lines, fmt.Sprintf("in %s %d %d: %v", sim.GetAddress(), dir, k, err))
				} else {
					su := in.SwapInfo.(StateUpdate)
					lines = append(lines, fmt.Sprintf("in %s %d %d: %s %s %s %d %s %s %d", sim.GetAddress(), dir, k,
						tokenAmount(in.TokenAmountIn), tokenAmount(in.Fee), tokenAmount(in.RemainingTokenAmountOut),
						in.Gas, su.Liquidity.Dec(), su.Price.Dec(), su.Tick))
				}
			}
		}
	}
	return lines
}

// TestQuoteGridUnchanged pins every quote field over quoteGrid to the digest the SDK word-by-word
// tick walk produced, so replacing the walk cannot move a quote, a gas count or an error.
func TestQuoteGridUnchanged(t *testing.T) {
	h := sha256.New()
	for _, line := range quoteGrid(t) {
		h.Write([]byte(line + "\n"))
	}
	require.Equal(t, "8f0b9200a53d22b80bcc786ba153dda80d5c13861dc3c5fc6a6a5bb5e2b4a62f", hex.EncodeToString(h.Sum(nil)))
}

func benchmarkCalcAmountOut(b *testing.B, sim *PoolSimulator, dir int, amount string) {
	tokens := sim.GetTokens()
	a, _ := new(big.Int).SetString(amount, 10)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[dir], Amount: a},
		TokenOut: tokens[1-dir]}
	if _, err := sim.CalcAmountOut(params); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		_, _ = sim.CalcAmountOut(params)
	}
}

// Full-range-only pool at tick 82476: the next tick is 887220 or -887220, thousands of words away.
func BenchmarkCalcAmountOutFullRange(b *testing.B) {
	benchmarkCalcAmountOut(b, ps, 0, "100000000000000")
}

func BenchmarkCalcAmountOutFullRangeUp(b *testing.B) {
	benchmarkCalcAmountOut(b, ps, 1, "1000000000")
}

// Thena USDT/WBNB with 29 ticks: a small swap and one crossing several ticks and gaps.
func BenchmarkCalcAmountOutThenaSmall(b *testing.B) {
	benchmarkCalcAmountOut(b, thenaPS, 1, "10000000000000000")
}

func BenchmarkCalcAmountOutThenaLarge(b *testing.B) {
	benchmarkCalcAmountOut(b, thenaPS, 1, "30000000000000000000")
}

// TestQuotesDoNotShareState guards the swap path's in-place math: UpdateBalance adopts SwapInfo's
// price and liquidity pointers, so a later quote must copy pool state rather than write into it.
func TestQuotesDoNotShareState(t *testing.T) {
	sim := thenaPS.CloneState().(*PoolSimulator)
	tokens := sim.GetTokens()
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[1], Amount: big.NewInt(3e18)}, TokenOut: tokens[0]}
	priceBefore, liquidityBefore := sim.globalState.Price.Clone(), sim.liquidity.Clone()

	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, priceBefore, sim.globalState.Price, "a quote must not move pool state")
	require.Equal(t, liquidityBefore, sim.liquidity)

	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	su := first.SwapInfo.(StateUpdate)
	adopted, adoptedLiquidity := su.Price.Clone(), su.Liquidity.Clone()
	_, err = sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, adopted, sim.globalState.Price, "a quote must not write into adopted state")
	require.Equal(t, adoptedLiquidity, sim.liquidity)
}
