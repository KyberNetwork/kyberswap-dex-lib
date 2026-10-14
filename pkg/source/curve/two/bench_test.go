package two

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Benchmarks compare the legacy math/big implementation (legacy_test.go) with the uint256 one on the CVX/ETH pool's
// on-chain state at block 26188397. Run with:
//
//	go test ./pkg/source/curve/two/ -run '^$' -bench . -benchmem
var benchSwaps = []struct {
	name   string
	i, j   int
	amount *big.Int
}{
	{"ETH-CVX_1ETH", 0, 1, big.NewInt(1e18)},
	{"CVX-ETH_1kCVX", 1, 0, new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))},
}

func benchImpls(b *testing.B) []struct {
	name string
	p    pool.IPoolSimulator
} {
	old, nw := newPair(b, cvxETH.entityPool())
	return []struct {
		name string
		p    pool.IPoolSimulator
	}{{"legacy", old}, {"uint256", nw}}
}

func benchParams(i, j int, amount *big.Int) pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: cvxETH.coins[i], Amount: amount}, TokenOut: cvxETH.coins[j]}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	now := cvxETH.timestamp
	pinClock(b, &now)
	for _, s := range benchSwaps {
		params := benchParams(s.i, s.j, s.amount)
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
	now := cvxETH.timestamp + 12
	pinClock(b, &now)
	for _, s := range benchSwaps {
		for _, impl := range benchImpls(b) {
			res, err := impl.p.CalcAmountOut(benchParams(s.i, s.j, s.amount))
			if err != nil {
				b.Fatal(err)
			}
			update := pool.UpdateBalanceParams{TokenAmountIn: benchParams(s.i, s.j, s.amount).TokenAmountIn,
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

// BenchmarkRouteStep mimics one path refresh in route finding: clone, quote, apply the swap, quote again.
func BenchmarkRouteStep(b *testing.B) {
	now := cvxETH.timestamp + 12
	pinClock(b, &now)
	for _, s := range benchSwaps {
		params := benchParams(s.i, s.j, s.amount)
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
