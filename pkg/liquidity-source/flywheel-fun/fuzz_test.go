package flywheelfun

import (
	"testing"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func FuzzCompositeQuotePurity(f *testing.F) {
	markets := snapshots(f)
	for i := range markets {
		for _, a := range []uint64{1, 100, 1000000000, 400000000000, 10000000000000} {
			f.Add(uint8(i), a)
		}
	}
	f.Fuzz(func(t *testing.T, index uint8, amount uint64) {
		p := markets[int(index)%len(markets)]
		s, err := NewPoolSimulatorWithBases(p, baseMap(t, p))
		if err != nil {
			t.Fatal(err)
		}
		params := pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: WETH, Amount: *uint256.NewInt(amount)}, TokenOut: p.Address}
		input := params.TokenAmountIn.Amount
		curve := s.Curve
		before := make([]string, len(s.allBases()))
		for i, b := range s.allBases() {
			before[i] = fingerprint(b)
		}
		first, e1 := s.CalcAmtOut(params)
		second, e2 := s.CalcAmtOut(params)
		if (e1 == nil) != (e2 == nil) || curve != s.Curve || !input.Eq(&params.TokenAmountIn.Amount) {
			t.Fatal("quote mutated state or input")
		}
		for i, b := range s.allBases() {
			if before[i] != fingerprint(b) {
				t.Fatal("route state mutated")
			}
		}
		if e1 != nil {
			return
		}
		if first.TokenAmountOut.Amount.Cmp(&second.TokenAmountOut.Amount) != 0 || first.TokenAmountOut.Amount.IsZero() || first.RemainingTokenAmountIn.Amount.Cmp(&input) > 0 {
			t.Fatal("invalid or nondeterministic quote")
		}
		cloned := s.CloneState().(*PoolSimulator)
		cloned.UpdateState(pool.UpdateStateParams{SwapInfo: first.SwapInfo})
		third, err := s.CalcAmtOut(params)
		if err != nil || third.TokenAmountOut.Amount.Cmp(&first.TokenAmountOut.Amount) != 0 {
			t.Fatal("clone update changed original")
		}
		info := first.SwapInfo.(SwapInfo)
		if info.Refunds != !first.RemainingTokenAmountIn.Amount.IsZero() {
			t.Fatal("Refunds must match the quoted refund")
		}
	})
}
