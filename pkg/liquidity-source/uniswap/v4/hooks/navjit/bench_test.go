package navjit

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := newTestSim(b)
	for _, bc := range []struct {
		name, in, out string
		amt           int64
	}{{"buy", simUsdg, simLot, 10_500_000_000}, {"sell", simLot, simUsdg, 5e18}} {
		params := pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: bc.in, Amount: big.NewInt(bc.amt)}, TokenOut: bc.out}
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.CalcAmountOut(params); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCloneStateUpdateBalance(b *testing.B) {
	sim := newTestSim(b)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: simUsdg, Amount: big.NewInt(1_000_000)}, TokenOut: simLot}
	res, err := sim.CalcAmountOut(params)
	if err != nil {
		b.Fatal(err)
	}
	upd := pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: simUsdg, Amount: *uint256.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmt{Token: res.TokenAmountOut.Token, Amount: *big256.FromBig(res.TokenAmountOut.Amount)},
		SwapInfo:       res.SwapInfo}
	b.ReportAllocs()
	for b.Loop() {
		sim.CloneState().UpdateState(upd)
	}
}
