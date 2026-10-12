package lotflow

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The advertised NAV-band limit, pinned exactly (ADV M03, M05-M07). Spec, computed here independently
// of priceLimit (no isqrt; a binary search over the sqrt price):
//   - the LOT price edge is offset = band-25 bps from NAV, rounded toward NAV: buys (zeroForOne on a
//     USDG-currency0 venue) floor nav*(1e4+offset)/1e4, sells ceil nav*(1e4-offset)/1e4;
//   - with K = 10^(36-usdgDecimals) * 2^192 the pool's LOT price at sqrt price L is K / L^2, so the
//     buy limit (price falling toward the edge from above) is the SMALLEST L with L^2*edge >= K, and the
//     sell limit the LARGEST L with L^2*edge <= K: both stay on the NAV side of the edge.

func expectedLimit(nav *big.Int, offset uint64, usdgDec int, buy bool) (*big.Int, *big.Int, *big.Int) {
	bps := big.NewInt(10_000)
	var edge *big.Int
	if buy {
		edge = new(big.Int).Quo(new(big.Int).Mul(nav, big.NewInt(int64(10_000+offset))), bps)
	} else {
		num := new(big.Int).Mul(nav, big.NewInt(int64(10_000-offset)))
		edge = new(big.Int).Quo(new(big.Int).Add(num, big.NewInt(9_999)), bps)
	}
	k := new(big.Int).Mul(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(36-usdgDec)), nil),
		new(big.Int).Lsh(big.NewInt(1), 192))
	f := func(l *big.Int) *big.Int { return new(big.Int).Mul(new(big.Int).Mul(l, l), edge) } // L^2 * edge
	lo, hi := big.NewInt(0), new(big.Int).Lsh(big.NewInt(1), 160)
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 { // invariant: f(lo) < K <= f(hi) for buys
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
		if (buy && f(mid).Cmp(k) >= 0) || (!buy && f(mid).Cmp(k) > 0) {
			hi = mid
		} else {
			lo = mid
		}
	}
	if buy {
		return hi, edge, k // smallest with f >= K
	}
	return lo, edge, k // largest with f <= K
}

func TestPriceLimitExact(t *testing.T) {
	n := 0
	live := map[Params]int{} // the live parameter sets, pinned at USDG decimals 6, both directions
	for _, path := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json",
		"testdata/a5/mag5-sell-floatBoundary-standing-first.json"} {
		h, _ := integrationHook(t, path)
		nav, err := newMarketState(h.Market).checkedNav()
		require.NoError(t, err)
		for _, band := range []uint64{h.Market.Venue.Params.BandBps, 226, 350, 1000} {
			for _, dec := range []int{6, 18, 0} {
				h.Market.Venue.Params.BandBps, h.Market.Venue.UsdgDecimals = band, dec
				offset := band - priceLimitBufferBps
				for _, buy := range []bool{true, false} {
					got := h.SqrtPriceLimit(buy) // UsdgIs0: zeroForOne == buy
					if band-priceLimitBufferBps <= max(h.Market.Venue.Params.BuySpreadBps,
						h.Market.Venue.Params.SellSpreadBps) {
						continue // spread guard refuses this band
					}
					require.NotNil(t, got, "%s band %d dec %d buy %v", path, band, dec, buy)
					want, edge, k := expectedLimit(nav, offset, dec, buy)
					require.Zero(t, want.Cmp(got.ToBig()), "%s band %d dec %d buy %v: limit %s, spec %s", path,
						band, dec, buy, got, want)
					// rounding direction, both sides: inside the band, and the 1-wei neighbour outside it
					l := got.ToBig()
					sq := func(x *big.Int) *big.Int { return new(big.Int).Mul(new(big.Int).Mul(x, x), edge) }
					next := new(big.Int).Add(l, big.NewInt(1))
					prev := new(big.Int).Sub(l, big.NewInt(1))
					if buy {
						require.GreaterOrEqual(t, sq(l).Cmp(k), 0, "buy limit price at or below the edge")
						require.Negative(t, sq(prev).Cmp(k), "1 wei lower is past the edge")
					} else {
						require.LessOrEqual(t, sq(l).Cmp(k), 0, "sell limit price at or above the edge")
						require.Positive(t, sq(next).Cmp(k), "1 wei higher is past the edge")
					}
					// closed bounds: a limit 1 bps looser or tighter is a different value
					looser, _, _ := expectedLimit(nav, offset+1, dec, buy)
					tighter, _, _ := expectedLimit(nav, offset-1, dec, buy)
					require.NotZero(t, looser.Cmp(l))
					require.NotZero(t, tighter.Cmp(l))
					if buy { // the buy limit falls as the band widens
						require.True(t, looser.Cmp(l) < 0 && l.Cmp(tighter) < 0)
					} else {
						require.True(t, looser.Cmp(l) > 0 && l.Cmp(tighter) > 0)
					}
					if dec == 6 {
						live[h.Market.Venue.Params]++
					}
					n++
				}
			}
		}
	}
	hook1 := Params{BuySpreadBps: 25, SellSpreadBps: 100, BandBps: 200, SizeBufferBps: 300, WidthTicks: 20}
	hook2 := Params{BuySpreadBps: 25, SellSpreadBps: 200, BandBps: 600, SizeBufferBps: 300, WidthTicks: 20}
	require.GreaterOrEqual(t, live[hook1], 2, "live hook #1 params 25/100/200, both directions")
	require.GreaterOrEqual(t, live[hook2], 2, "live hook #2 params 25/200/600, both directions")
	t.Logf("%d limits pinned (live hook #1 set %d, hook #2 set %d)", n, live[hook1], live[hook2])
	require.Greater(t, n, 40)
}

// The limit-refusal path end to end through the complete v4 pool simulator (no live sample reaches it,
// ADV): on the 5-constituent venue a $300k buy fills completely without a price limit, but the
// executor's NAV-band limit would cut it short, so it is refused as ErrPartialFill, never quoted as a
// partial; $100k on the same state fills inside the limit.
func TestPriceLimitRefusalEndToEnd(t *testing.T) {
	h, _ := integrationHook(t, "testdata/a5/mag5-sell-floatBoundary-standing-first.json")
	usd := func(k int64) *big.Int { return new(big.Int).Mul(big.NewInt(k), pow10(h.Market.Venue.UsdgDecimals)) }
	big300k := usd(300_000)
	free, err := QuoteConservative(h.cfg(), h.Market, SwapInput{BuyLot: true, AmountIn: big300k, Prefund: true})
	require.NoError(t, err)
	require.Zero(t, free.Venue.AmountIn.Cmp(big300k), "without a price limit the buy fills completely")
	limit := h.SqrtPriceLimit(h.Market.Venue.UsdgIs0)
	require.NotNil(t, limit)
	capped, err := QuoteConservative(h.cfg(), h.Market, SwapInput{BuyLot: true, AmountIn: big300k, Prefund: true,
		SqrtPriceLimitX96: limit.ToBig()})
	require.NoError(t, err)
	require.Negative(t, capped.Venue.AmountIn.Cmp(big300k), "the NAV-band limit cuts the venue swap short")

	sim, err := pool.Factory(uniswapv4.DexType)(pool.FactoryParams{EntityPool: integrationPool(t, h),
		ChainID: valueobject.ChainIDRobinhood})
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: h.Usdg, Amount: big300k}, TokenOut: h.Lot})
	require.ErrorIs(t, err, ErrPartialFill)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: h.Usdg, Amount: usd(100_000)}, TokenOut: h.Lot})
	require.NoError(t, err)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())
	t.Logf("$300k: unlimited fill, limited venue input %s of %s -> ErrPartialFill; $100k fills %s LOT wei",
		capped.Venue.AmountIn, big300k, res.TokenAmountOut.Amount)
}
