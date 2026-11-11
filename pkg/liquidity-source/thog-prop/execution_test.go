package thogprop

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestExecutionFrictionStrictThresholdsAndFreshness(t *testing.T) {
	for _, test := range []struct{ base, gas, raw uint64 }{
		{100, 0, 0}, {100, 150, 0}, {100, 151, 600}, {100, 200, 600}, {100, 201, 800}, {100, 250, 800}, {100, 251, 1200},
		{3, 4, 0}, {3, 5, 600}, {3, 6, 600}, {3, 7, 800}, {3, 8, 1200}, {0, 0, 0}, {0, 1, 1200},
	} {
		for _, warm := range []bool{false, true} {
			for _, fresh := range []bool{false, true} {
				want := test.raw
				if warm {
					want = 1200
				}
				if fresh && want != 0 {
					want = want/4 - 100
				}
				ctx := ExecutionContext{BaseFee: u(test.base), GasPrice: u(test.gas), AuctionHandlerWarm: warm}
				require.Equal(t, want, ctx.friction(fresh), "base=%d gas=%d warm=%t fresh=%t", test.base, test.gas, warm, fresh)
			}
		}
	}
}
func TestExecutionFrictionDoesNotOverflowAtUint256Boundary(t *testing.T) {
	var half, third uint256.Int
	half.Rsh(&maxUint, 1)
	three := u(3)
	third.Div(&maxUint, &three)
	values := []uint256.Int{u(0), u(1), half, third, maxUint}
	for _, base := range values {
		for _, gas := range values {
			// Independent unbounded arithmetic verifies strict rational thresholds.
			g, b := gas.ToBig(), base.ToBig()
			g2 := new(big.Int).Mul(g, big.NewInt(2))
			var want uint64
			switch {
			case g2.Cmp(new(big.Int).Mul(b, big.NewInt(5))) > 0:
				want = 1200
			case g.Cmp(new(big.Int).Mul(b, big.NewInt(2))) > 0:
				want = 800
			case g2.Cmp(new(big.Int).Mul(b, big.NewInt(3))) > 0:
				want = 600
			}
			ctx := ExecutionContext{BaseFee: base, GasPrice: gas}
			require.Equal(t, want, ctx.friction(false))
		}
	}
}

func TestGasEstimatesCoverModel3HarnessMeasurements(t *testing.T) {
	var maximum [2]int64
	for _, snapshot := range fixture(t, "execution-model3.json.gz").Snapshots {
		s := stateFrom(t, snapshot)
		sim, err := NewPoolSimulator(entityFrom(t, s))
		require.NoError(t, err)
		for _, c := range snapshot.Cases {
			if !c.Success || c.Method != "execution" {
				continue
			}
			require.Greater(t, c.MeasuredGas, int64(0))
			require.LessOrEqual(t, c.MeasuredGas, sim.estimateGas(c.InputIndex, c.OutputIndex))
			group := 0
			if s.Tokens[c.InputIndex].Category != s.Tokens[c.OutputIndex].Category {
				group = 1
			}
			if c.MeasuredGas > maximum[group] {
				maximum[group] = c.MeasuredGas
			}
		}
	}
	require.Positive(t, maximum[0])
	require.Positive(t, maximum[1])
	t.Logf("Model-3 harness maximums: same category %d, cross category %d", maximum[0], maximum[1])
}
