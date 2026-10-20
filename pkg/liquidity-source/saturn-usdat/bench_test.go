package saturnusdat

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := newTestPool(&testing.T{}, nil)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDC, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      USDat,
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := sim.CalcAmountOut(params); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateBalance(b *testing.B) {
	sim := newTestPool(&testing.T{}, nil)
	res, _ := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: USDC, Amount: big.NewInt(1)}, TokenOut: USDat})
	b.ReportAllocs()
	for b.Loop() {
		sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	}
}

func BenchmarkCloneState(b *testing.B) {
	sim := newTestPool(&testing.T{}, nil)
	b.ReportAllocs()
	for b.Loop() {
		_ = sim.CloneState()
	}
}
