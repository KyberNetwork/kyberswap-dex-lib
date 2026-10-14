package tricrypto

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Benchmarks compare the legacy math/big implementation (legacy_test.go) with the uint256 one on Curve tricrypto2's
// on-chain state at block 26181452. Run with:
//
//	go test ./pkg/source/curve/tricrypto/ -run '^$' -bench . -benchmem
var benchSwaps = []struct {
	name    string
	in, out string
	amount  int64
}{
	{"WETH-USDT_4ETH", tri2WETH, tri2USDT, 4e18},
	{"USDT-WETH_10k", tri2USDT, tri2WETH, 10_000e6},
	{"WBTC-WETH_1BTC", tri2WBTC, tri2WETH, 1e8},
}

func benchImpls(b *testing.B) []struct {
	name string
	p    pool.IPoolSimulator
} {
	old, nw := newPair(b, tri2AtBlock26181452.entityPool())
	return []struct {
		name string
		p    pool.IPoolSimulator
	}{{"legacy", old}, {"uint256", nw}}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for _, s := range benchSwaps {
		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: s.in, Amount: big.NewInt(s.amount)}, TokenOut: s.out}
		for _, impl := range benchImpls(b) {
			b.Run(s.name+"/"+impl.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := impl.p.CalcAmountOut(params); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkCloneState(b *testing.B) {
	for _, impl := range benchImpls(b) {
		b.Run(impl.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = impl.p.CloneState()
			}
		})
	}
}

// BenchmarkUpdateBalance measures CloneState + UpdateBalance (exchange + tweak_price), which is what route
// refreshes do for every path through the pool.
func BenchmarkUpdateBalance(b *testing.B) {
	now := int64(tri2Block53Timestamp)
	pinClock(b, &now)
	for _, s := range benchSwaps {
		for _, impl := range benchImpls(b) {
			res, err := impl.p.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: s.in, Amount: big.NewInt(s.amount)}, TokenOut: s.out})
			if err != nil {
				b.Fatal(err)
			}
			update := pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: s.in, Amount: big.NewInt(s.amount)},
				TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee}
			b.Run(s.name+"/"+impl.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					impl.p.CloneState().UpdateBalance(update)
				}
			})
		}
	}
}

// BenchmarkRouteStep mimics one path refresh in the hill climber: clone, quote, apply the swap, quote again.
func BenchmarkRouteStep(b *testing.B) {
	now := int64(tri2Block53Timestamp)
	pinClock(b, &now)
	for _, s := range benchSwaps {
		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: s.in, Amount: big.NewInt(s.amount)}, TokenOut: s.out}
		for _, impl := range benchImpls(b) {
			b.Run(s.name+"/"+impl.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					p := impl.p.CloneState()
					res, err := p.CalcAmountOut(params)
					if err != nil {
						b.Fatal(err)
					}
					p.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn,
						TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee})
					if _, err = p.CalcAmountOut(params); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
