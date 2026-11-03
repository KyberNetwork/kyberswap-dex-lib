package thogprop

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// benchSim returns a simulator on the first execution fixture and one successful
// quote request on it (with and without an xStock leg).
func benchSim(b *testing.B, xStock bool) (*PoolSimulator, pool.CalcAmountOutParams) {
	b.Helper()
	f := fixture(b, "execution-model3.json.gz")
	s := stateFrom(b, f.Snapshots[0])
	sim, err := NewPoolSimulator(entityFrom(b, s))
	require.NoError(b, err)
	for _, c := range f.Snapshots[0].Cases {
		if c.Method != "execution" || !c.Success {
			continue
		}
		in, out := s.Tokens[c.InputIndex].Address, s.Tokens[c.OutputIndex].Address
		if (isXStock(in) || isXStock(out)) != xStock {
			continue
		}
		amount := number(b, c.Amount)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount.ToBig()}, TokenOut: out}
		if _, err := sim.quoteAtCapture(params); err == nil {
			return sim, params
		}
	}
	b.Fatal("no successful case")
	return nil, pool.CalcAmountOutParams{}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for _, xStock := range []bool{false, true} {
		name := map[bool]string{false: "plain", true: "xstock"}[xStock]
		b.Run(name, func(b *testing.B) {
			sim, params := benchSim(b, xStock)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.calcAmountOut(params, sim.capturedAtMs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCloneAndUpdateBalance(b *testing.B) {
	sim, params := benchSim(b, true)
	r, err := sim.quoteAtCapture(params)
	require.NoError(b, err)
	update := pool.UpdateBalanceParams{SwapInfo: r.SwapInfo}
	b.ReportAllocs()
	for b.Loop() {
		sim.CloneState().(*PoolSimulator).UpdateBalance(update)
	}
}
