package flywheelfun

import (
	"testing"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func BenchmarkCalcAmtOut(b *testing.B) {
	for _, p := range snapshots(b) {
		s := newSim(b, p)
		in := pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: WETH, Amount: *uint256.NewInt(1e9)}, TokenOut: s.Info.Address}
		r, err := s.CalcAmtOut(in)
		if err != nil {
			continue
		}
		out := pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: s.Info.Address, Amount: r.TokenAmountOut.Amount}, TokenOut: WETH}
		b.Run(s.Info.Address[:8]+"/buy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = s.CalcAmtOut(in)
			}
		})
		b.Run(s.Info.Address[:8]+"/sell", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = s.CalcAmtOut(out)
			}
		})
		b.Run(s.Info.Address[:8]+"/clone", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = s.CloneState()
			}
		})
	}
}
