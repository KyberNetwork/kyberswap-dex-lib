package machima

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// calcAmtInGolden is one exact-out quote captured from the legacy big.Int CalcAmountIn before the native port.
// Variant names the pool built by calcAmtInVariants.
type calcAmtInGolden struct {
	Variant   string `json:"variant"`
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	AmountOut string `json:"amountOut"`
	WantIn    string `json:"wantIn,omitempty"`
	WantErr   string `json:"wantErr,omitempty"`
}

var calcAmtInVariants = map[string]func(t *testing.T) *PoolSimulator{
	"plain":    plainSim,
	"taxed":    taxedSim,
	"xmaTaxed": xmaTaxedSim,
	"xmaFloor": xmaFloorSim,
}

func loadCalcAmtInGoldens(t *testing.T) []calcAmtInGolden {
	t.Helper()
	b, err := os.ReadFile("testdata/calc_amount_in_golden.json")
	require.NoError(t, err)
	var goldens []calcAmtInGolden
	require.NoError(t, json.Unmarshal(b, &goldens))
	require.NotEmpty(t, goldens)
	return goldens
}

// TestCalcAmtInParityWithLegacy checks native CalcAmtIn against the legacy CalcAmountIn goldens, errors included.
func TestCalcAmtInParityWithLegacy(t *testing.T) {
	sims := make(map[string]*PoolSimulator, len(calcAmtInVariants))
	for name, build := range calcAmtInVariants {
		sims[name] = build(t)
	}
	for _, g := range loadCalcAmtInGoldens(t) {
		t.Run(g.Variant+"/"+g.TokenIn[len(g.TokenIn)-4:]+"->"+g.TokenOut[len(g.TokenOut)-4:]+"/"+g.AmountOut, func(t *testing.T) {
			var out uint256.Int
			require.NoError(t, out.SetFromDecimal(g.AmountOut))
			res, err := sims[g.Variant].CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: g.TokenOut, Amount: out},
				TokenIn:        g.TokenIn,
			})
			if g.WantErr != "" {
				require.EqualError(t, err, g.WantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, g.WantIn, res.TokenAmountIn.Amount.Dec(), "amount in")
		})
	}
}

// TestCalcAmountInIPoolSim pins native CalcAmtIn with the shared helper. Index 0 is WETH and 1 is the launched token,
// so 0->1 is a buy and 1->0 a sell. The values are the legacy CalcAmountIn goldens from testdata.
func TestCalcAmtInIPoolSim(t *testing.T) {
	testutil.TestCalcAmountInIPoolSim(t, taxedSim(t), map[int]map[int]map[string]string{
		0: {1: {"10": "13"}},
		1: {0: {"10": "13"}},
	})
	testutil.TestCalcAmountInIPoolSim(t, plainSim(t), map[int]map[int]map[string]string{
		0: {1: {"10": "12"}},
		1: {0: {"10": "12"}},
	})
}

// TestCalcAmtInIsTaxAware is the exact-out tax regression test: an untaxed CalcAmtIn would be ~5% short, so the
// taxed input must exceed the untaxed one by the tax rate.
func TestCalcAmtInIsTaxAware(t *testing.T) {
	for _, tc := range []struct {
		name              string
		tokenIn, tokenOut string
	}{
		{"buy", wethAddr, tokenAddr},
		{"sell", tokenAddr, wethAddr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out uint256.Int
			out.SetUint64(1e14)
			taxed, err := taxedSim(t).CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: tc.tokenOut, Amount: out},
				TokenIn:        tc.tokenIn,
			})
			require.NoError(t, err)
			untaxed, err := plainSim(t).CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: tc.tokenOut, Amount: out},
				TokenIn:        tc.tokenIn,
			})
			require.NoError(t, err)
			assert.Greater(t, taxed.TokenAmountIn.Amount.Cmp(&untaxed.TokenAmountIn.Amount), 0,
				"taxed exact-out input must exceed the untaxed input")
		})
	}
}

// TestCalcAmtInRoundTrip checks that feeding the native amountIn back through native CalcAmtOut clears the requested
// output, and does not overshoot by more than 0.1% (plus 1 wei of integer rounding). An untaxed CalcAmtIn fails the
// overshoot bound. The XMA launch floor caps what the pool can deliver, so those cases are skipped: the legacy quote
// returns the same capped input for every larger output, and the parity test pins that.
func TestCalcAmtInRoundTrip(t *testing.T) {
	sims := make(map[string]*PoolSimulator, len(calcAmtInVariants))
	for name, build := range calcAmtInVariants {
		sims[name] = build(t)
	}
	checked := 0
	for _, g := range loadCalcAmtInGoldens(t) {
		if g.WantErr != "" || g.Variant == "xmaFloor" && g.TokenIn == xmaAddr {
			continue
		}
		t.Run(g.Variant+"/"+g.TokenIn[len(g.TokenIn)-4:]+"->"+g.TokenOut[len(g.TokenOut)-4:]+"/"+g.AmountOut, func(t *testing.T) {
			sim := sims[g.Variant]
			var want uint256.Int
			require.NoError(t, want.SetFromDecimal(g.AmountOut))
			in, err := sim.CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: g.TokenOut, Amount: want},
				TokenIn:        g.TokenIn,
			})
			require.NoError(t, err)
			back, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: g.TokenIn, Amount: in.TokenAmountIn.Amount},
				TokenOut:      g.TokenOut,
			})
			require.NoError(t, err)
			got := back.TokenAmountOut.Amount
			assert.GreaterOrEqual(t, got.Cmp(&want), 0, "round trip undershot")
			upper := new(big.Int).Div(new(big.Int).Mul(want.ToBig(), big.NewInt(10010)), big.NewInt(10000))
			upper.Add(upper, big.NewInt(1))
			assert.LessOrEqual(t, got.ToBig().Cmp(upper), 0, "round trip overshot")
		})
		checked++
	}
	require.NotZero(t, checked)
}
