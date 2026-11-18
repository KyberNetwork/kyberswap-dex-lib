package thogprop

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The current registry includes all nine stocks and a larger covariance matrix.
// Keep the fixture, amount and quote block fixed across before/after comparisons.
func BenchmarkCurrentRegistry(b *testing.B) {
	f := fixture(b, "deployed-stored-multipliers.json.gz")
	state := stateFrom(b, f.Snapshots[0])
	for _, pair := range []struct {
		name          string
		input, output int
	}{
		{"USDC_WETH", 0, 4}, {"WETH_USDC", 4, 0},
		{"USDC_SMHx", 0, 13}, {"MUx_USDC", 15, 0}, {"MUx_SKHYx", 15, 16},
	} {
		b.Run(pair.name, func(b *testing.B) {
			sim, err := NewPoolSimulator(entityFrom(b, state))
			require.NoError(b, err)
			var params pool.CalcAmountOutParams
			for _, c := range f.Snapshots[0].Cases {
				if c.Method != "quote" || !c.Success || c.InputIndex != pair.input || c.OutputIndex != pair.output {
					continue
				}
				amount := number(b, c.Amount)
				candidate := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: state.Tokens[pair.input].Address, Amount: amount.ToBig()}, TokenOut: state.Tokens[pair.output].Address}
				if _, err := sim.quoteAtCapture(candidate); err == nil {
					params = candidate
					break
				}
			}
			require.NotNil(b, params.TokenAmountIn.Amount, "fixture must contain an executable pair")
			b.ReportAllocs()
			for b.Loop() {
				if _, err := sim.calcAmountOut(params, sim.capturedAtMs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
