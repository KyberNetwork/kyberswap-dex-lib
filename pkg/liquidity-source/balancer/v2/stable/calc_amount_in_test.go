package stable

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// calcAmountInGolden is one case from testdata/calc_amount_in_golden.json, captured from the legacy CalcAmountIn.
// Pool indexes the pools array. An empty WantErr means the quote succeeds with WantIn and WantFee.
type calcAmountInGolden struct {
	Pool      int    `json:"pool"`
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	AmountOut string `json:"amountOut"`
	WantIn    string `json:"wantIn,omitempty"`
	WantFee   string `json:"wantFee,omitempty"`
	WantErr   string `json:"wantErr,omitempty"`
}

type calcAmountInGoldenFile struct {
	Pools []json.RawMessage    `json:"pools"`
	Cases []calcAmountInGolden `json:"cases"`
}

func loadCalcAmountInGolden(t *testing.T) (calcAmountInGoldenFile, []*PoolSimulator) {
	t.Helper()
	raw, err := os.ReadFile("testdata/calc_amount_in_golden.json")
	require.NoError(t, err)
	var f calcAmountInGoldenFile
	require.NoError(t, json.Unmarshal(raw, &f))

	sims := make([]*PoolSimulator, len(f.Pools))
	for i, p := range f.Pools {
		var ep entity.Pool
		require.NoError(t, json.Unmarshal(p, &ep))
		sims[i], err = NewPoolSimulator(ep, nil)
		require.NoError(t, err)
	}
	return f, sims
}

// TestCalcAmtInParityWithLegacy checks that native CalcAmtIn returns exactly the legacy CalcAmountIn golden,
// including the error message, for every pool, token pair, and amount in the sweep.
func TestCalcAmtInParityWithLegacy(t *testing.T) {
	t.Parallel()
	f, sims := loadCalcAmountInGolden(t)
	require.NotEmpty(t, f.Cases)

	for _, c := range f.Cases {
		var out uint256.Int
		require.NoError(t, out.SetFromDecimal(c.AmountOut))
		res, err := sims[c.Pool].CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: c.TokenOut, Amount: out},
			TokenIn:        c.TokenIn,
		})
		if c.WantErr != "" {
			require.EqualErrorf(t, err, c.WantErr, "pool=%d %s->%s out=%s", c.Pool, c.TokenIn, c.TokenOut, c.AmountOut)
			continue
		}
		require.NoErrorf(t, err, "pool=%d %s->%s out=%s", c.Pool, c.TokenIn, c.TokenOut, c.AmountOut)
		require.Equalf(t, c.WantIn, res.TokenAmountIn.Amount.String(), "amountIn pool=%d out=%s", c.Pool, c.AmountOut)
	}
}

// feeBps converts an 18-decimal fee (1e18 = 100%) to whole basis points.
func feeBps(fee *uint256.Int) uint64 {
	var bps uint256.Int
	bps.Mul(fee, uint256.NewInt(10_000))
	bps.Div(&bps, uint256.NewInt(1e18))
	return bps.Uint64()
}

// TestCalcAmtInRoundTrip checks that quoting the native CalcAmtIn input forward returns the requested output within
// the pool's swap fee plus 15 bps of rounding slack. Pools with dust outputs (below 1e-6 of the out-token reserve)
// are skipped, since integer rounding then dominates.
func TestCalcAmtInRoundTrip(t *testing.T) {
	t.Parallel()
	f, sims := loadCalcAmountInGolden(t)
	const slackBps = 15

	checked := 0
	for _, c := range f.Cases {
		if c.WantErr != "" || c.TokenIn == c.TokenOut {
			continue
		}
		var out uint256.Int
		require.NoError(t, out.SetFromDecimal(c.AmountOut))
		sim := sims[c.Pool]
		floor := new(big.Int).Quo(sim.Info.Reserves[sim.GetTokenIndex(c.TokenOut)], big.NewInt(1_000_000))
		if out.IsZero() || out.ToBig().Cmp(floor) < 0 {
			continue
		}
		in, err := sim.CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: c.TokenOut, Amount: out},
			TokenIn:        c.TokenIn,
		})
		require.NoError(t, err)

		fwd, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: c.TokenIn, Amount: in.TokenAmountIn.Amount},
			TokenOut:      c.TokenOut,
		})
		require.NoErrorf(t, err, "forward quote pool=%d in=%s", c.Pool, in.TokenAmountIn.Amount.String())

		var gap, bps uint256.Int
		if fwd.TokenAmountOut.Amount.Gt(&out) {
			gap.Sub(&fwd.TokenAmountOut.Amount, &out)
		} else {
			gap.Sub(&out, &fwd.TokenAmountOut.Amount)
		}
		bps.Mul(&gap, uint256.NewInt(10_000))
		bps.Div(&bps, &out)
		tol := feeBps(sim.swapFeePercentage) + slackBps
		require.Truef(t, bps.Cmp(uint256.NewInt(tol)) <= 0, "round trip gap %s bps > %d pool=%d out=%s",
			bps.String(), tol, c.Pool, c.AmountOut)
		checked++
	}
	require.NotZero(t, checked, "round trip must cover at least one successful exact-out quote")
}
