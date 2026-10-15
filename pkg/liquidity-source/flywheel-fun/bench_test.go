package flywheelfun

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func BenchmarkCalcAmountOut(b *testing.B) {
	for _, p := range snapshots(b) {
		s := newSim(b, p)
		in := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(1e9)}, TokenOut: s.Info.Address}
		r, err := s.CalcAmountOut(in)
		if err != nil {
			continue
		}
		out := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Info.Address, Amount: r.TokenAmountOut.Amount}, TokenOut: WETH}
		b.Run(s.Info.Address[:8]+"/buy", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = s.CalcAmountOut(in)
			}
		})
		b.Run(s.Info.Address[:8]+"/sell", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = s.CalcAmountOut(out)
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
