package uniswapv4

import (
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// newTokenListsTestPool builds a pool of two of the tokens in can_swap_wrap_test.go (by symbol).
func newTokenListsTestPool(t testing.TB, token0, token1 string) *PoolSimulator {
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "uniswapV4PoolAddress",
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: tokens[token0].Address, Swappable: true},
			{Address: tokens[token1].Address, Swappable: true},
		},
		StaticExtra: `{"0x0":[false,false],"fee":100,"tS":1,"hooks":"0x0000000000000000000000000000000000000000"}`,
		Extra: `{"liquidity":1000000000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":1,"tick":0,` +
			`"ticks":[{"index":-887272,"liquidityGross":1000000000000000000,"liquidityNet":1000000000000000000},` +
			`{"index":887272,"liquidityGross":1000000000000000000,"liquidityNet":-1000000000000000000}]}`,
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return sim
}

var tokenListsTestPools = [][2]string{{"WETH", "KNC"}, {"UNI", "fwUNI"}, {"fwWETH", "fwUNI"}}

// The cached GetTokens/CanSwapTo/CanSwapFrom must answer exactly what computing them answers, for
// every token: the pool's tokens, the tokens they unwrap to, and unrelated tokens (not cached).
func TestTokenListsMatchComputed(t *testing.T) {
	t.Parallel()
	for _, pair := range tokenListsTestPools {
		sim := newTokenListsTestPool(t, pair[0], pair[1])
		require.Equal(t, sim.computeTokens(), sim.GetTokens(), pair)
		for _, token := range tokens {
			want := sim.computeCanSwapTo(token.Address)
			for _, got := range [][]string{sim.CanSwapTo(token.Address), sim.CanSwapFrom(token.Address)} {
				require.Equal(t, want == nil, got == nil, "%v %s", pair, token.Address)
				require.ElementsMatch(t, want, got, "%v %s", pair, token.Address)
			}
		}
	}
}

// The route finder calls these for every pool on every hop, so they must not allocate for any
// token of GetTokens, the only tokens it asks a pool about.
func TestTokenListsDoNotAllocate(t *testing.T) {
	for _, pair := range tokenListsTestPools {
		sim := newTokenListsTestPool(t, pair[0], pair[1])
		allocs := testing.AllocsPerRun(100, func() {
			for _, token := range sim.GetTokens() {
				_ = sim.CanSwapTo(token)
				_ = sim.CanSwapFrom(token)
			}
		})
		require.Zero(t, allocs, pair)
	}
}

// A clone, and a clone that has swapped, answer the same as the original.
func TestTokenListsSurviveCloneAndSwap(t *testing.T) {
	t.Parallel()
	sim := newTokenListsTestPool(t, "fwWETH", "fwUNI")
	clone := sim.CloneState().(*PoolSimulator)
	amountIn := pool.TokenAmount{Token: tokens["fwWETH"].Address, Amount: big.NewInt(1e15)}
	res, err := clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: amountIn, TokenOut: tokens["fwUNI"].Address})
	require.NoError(t, err)
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: amountIn, TokenAmountOut: *res.TokenAmountOut,
		Fee: *res.Fee, SwapInfo: res.SwapInfo})
	require.Equal(t, sim.GetTokens(), clone.GetTokens())
	for _, token := range sim.GetTokens() {
		require.Equal(t, sim.CanSwapTo(token), clone.CanSwapTo(token))
	}
	require.True(t, slices.Contains(clone.CanSwapFrom(tokens["WETH"].Address), tokens["UNI"].Address))
}

func BenchmarkTokenLists(b *testing.B) {
	for _, pair := range [][2]string{{"WETH", "KNC"}, {"fwWETH", "fwUNI"}} {
		sim := newTokenListsTestPool(b, pair[0], pair[1])
		poolTokens := sim.GetTokens()
		b.Run(pair[0]+"-"+pair[1]+"/cached", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = sim.GetTokens()
				for _, token := range poolTokens {
					_ = sim.CanSwapTo(token)
				}
			}
		})
		b.Run(pair[0]+"-"+pair[1]+"/computed", func(b *testing.B) { // what every call cost before the cache
			b.ReportAllocs()
			for b.Loop() {
				_ = sim.computeTokens()
				for _, token := range poolTokens {
					_ = sim.computeCanSwapTo(token)
				}
			}
		})
	}
}
