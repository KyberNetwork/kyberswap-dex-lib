package ilyrislb

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// liveSim loads the WETH/USDG pool (196 bins) captured from Robinhood block 81899939.
func liveSim(tb testing.TB) *PoolSimulator {
	tb.Helper()
	raw, err := os.ReadFile("testdata/pool_weth_usdg.json")
	if err != nil {
		tb.Fatal(err)
	}
	var ep entity.Pool
	if err := json.Unmarshal(raw, &ep); err != nil {
		tb.Fatal(err)
	}
	s, err := NewPoolSimulator(ep)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

func benchQuote(b *testing.B, xForY bool, amount int64) {
	s := liveSim(b)
	in, out := s.Info.Tokens[1], s.Info.Tokens[0]
	if xForY {
		in, out = out, in
	}
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(amount)}, TokenOut: out}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.CalcAmountOut(params); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalcAmountOut1Bin(b *testing.B)   { benchQuote(b, true, 1e13) }
func BenchmarkCalcAmountOut55Bins(b *testing.B) { benchQuote(b, true, 3e15) }
func BenchmarkCalcAmountOutYForX(b *testing.B)  { benchQuote(b, false, 8_000_000) }

func BenchmarkCloneStateUpdateBalance(b *testing.B) {
	s := liveSim(b)
	in := pool.TokenAmount{Token: s.Info.Tokens[0], Amount: big.NewInt(3e15)}
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: s.Info.Tokens[1]})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		c := s.CloneState()
		c.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
	}
}
