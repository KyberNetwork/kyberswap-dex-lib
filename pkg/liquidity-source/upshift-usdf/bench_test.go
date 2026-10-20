package upshiftusdf

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := newSim(b, staleExtra(), 1792142262)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDC, Amount: big.NewInt(1_000_000_000)}, TokenOut: USDF}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := sim.CalcAmountOut(params); err != nil {
			b.Fatal(err)
		}
	}
}
