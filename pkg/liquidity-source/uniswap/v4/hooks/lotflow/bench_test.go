package lotflow

import (
	"context"
	"math/big"
	"testing"
	"time"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Quote cost (AUDIT F4; no optimisation in v1, numbers in README "Performance"). Each quote runs the
// hook's swap on math/big state: legs over every constituent, candidate probing, NAV check.
var benchVenues = []struct{ name, path string }{
	{"stock-2const", "testdata/fixtures/navjit2-buy-1kusd.json"},
	{"stock-5const", "testdata/a5/mag5-sell-floatBoundary-standing-first.json"},
	{"mq-3const", "testdata/mq/mq-buy-1kusd.json"},
}

func benchHook(b *testing.B, path string) *Hook {
	b.Helper()
	c, err := LoadCase(path)
	if err != nil {
		b.Fatal(err)
	}
	t := &testing.T{}
	h := hookOf(t, c, marketOf(t, c))
	h.TrackedAt = time.Now().Unix()
	return h
}

func benchSides(h *Hook) []struct {
	side   string
	params *uniswapv4.BeforeSwapParams
} {
	usdgIs0 := h.Market.Venue.UsdgIs0
	return []struct {
		side   string
		params *uniswapv4.BeforeSwapParams
	}{
		{"buy-1kusd", &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: usdgIs0,
			AmountSpecified: new(big.Int).Mul(big.NewInt(1_000), pow10(h.Market.Venue.UsdgDecimals))}},
		{"sell-10lot", &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: !usdgIs0,
			AmountSpecified: new(big.Int).Mul(big.NewInt(10), bigE18)}},
	}
}

func BenchmarkBeforeSwap(b *testing.B) {
	for _, v := range benchVenues {
		h := benchHook(b, v.path)
		for _, s := range benchSides(h) {
			if _, err := h.BeforeSwap(s.params); err != nil {
				b.Logf("%s %s refused (%v): not benchmarked", v.name, s.side, err)
				continue
			}
			b.Run(v.name+"/"+s.side, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := h.BeforeSwap(s.params); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkCalcAmountOut is the full v4 pool simulator around the hook, the level #1740's
// navjit BenchmarkCalcAmountOut measures.
func BenchmarkCalcAmountOut(b *testing.B) {
	for _, v := range benchVenues {
		h := benchHook(b, v.path)
		sim, err := pool.Factory(uniswapv4.DexType)(pool.FactoryParams{EntityPool: integrationPool(&testing.T{}, h),
			ChainID: valueobject.ChainIDRobinhood})
		if err != nil {
			b.Fatal(err)
		}
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: h.Usdg,
			Amount: new(big.Int).Mul(big.NewInt(1_000), pow10(h.Market.Venue.UsdgDecimals))}, TokenOut: h.Lot}
		if _, err := sim.CalcAmountOut(params); err != nil {
			b.Logf("%s refused (%v): not benchmarked", v.name, err)
			continue
		}
		b.Run(v.name+"/buy-1kusd", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.CalcAmountOut(params); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetReserves(b *testing.B) {
	for _, v := range benchVenues {
		h := benchHook(b, v.path)
		b.Run(v.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := h.GetReserves(context.Background(), nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
