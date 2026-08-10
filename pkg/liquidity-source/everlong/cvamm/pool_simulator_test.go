package everlongcvamm

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	testALM     = "0x00000000000000000000000000000000000cva77"
	testStable  = "0x0000000000000000000000000000000000000001"
	testVol     = "0x0000000000000000000000000000000000000002"
	testFeeSIn  = 400000000000000  // 4 bps, stable-in
	testFeeVIn  = 1100000000000000 // 11 bps, volatile-in
	hugeReserve = "100000000000000000000000000000000000000"
)

// simFromFixture builds a PoolSimulator on a fixture state. Reserves default to
// effectively-unbounded so the solvency clamp does not bind unless a test wants it to.
func simFromFixture(t *testing.T, c fixtureCase, reserves entity.PoolReserves) *PoolSimulator {
	t.Helper()
	extraBytes, err := json.Marshal(Extra{
		Support:          c.sup,
		XWad:             c.x,
		AnchorSqrtX96:    c.anchor,
		Kappa:            c.kappa,
		FeeStableInWad:   uint256.NewInt(testFeeSIn),
		FeeVolatileInWad: uint256.NewInt(testFeeVIn),
	})
	require.NoError(t, err)
	if reserves == nil {
		reserves = entity.PoolReserves{hugeReserve, hugeReserve}
	}
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  testALM,
		Exchange: "everlong-cvamm",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testStable, Swappable: true},
			{Address: testVol, Swappable: true},
		},
		Reserves:    reserves,
		StaticExtra: "{}",
		Extra:       string(extraBytes),
	})
	require.NoError(t, err)
	return sim
}

// fillableCases picks fixture cases that actually fill (used > 0, gross > 0), keyed by
// direction, so behavior tests exercise real fills on chain-derived ground truth.
func fillableCases(t *testing.T) (stableIn, volatileIn []fixtureCase) {
	t.Helper()
	swaps, _ := loadFixtures(t)
	for _, c := range swaps {
		var used uint256.Int
		used.Sub(c.amountIn, c.unspent)
		if used.IsZero() || c.gross.IsZero() {
			continue
		}
		if c.stableIn {
			stableIn = append(stableIn, c)
		} else {
			volatileIn = append(volatileIn, c)
		}
	}
	require.NotEmpty(t, stableIn)
	require.NotEmpty(t, volatileIn)
	return
}

func calc(sim *PoolSimulator, c fixtureCase) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := testStable, testVol
	if !c.stableIn {
		tokenIn, tokenOut = testVol, testStable
	}
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: c.amountIn.ToBig()},
		TokenOut:      tokenOut,
	})
}

// TestCalcAmountOutMatchesFixtures replays every fillable fixture case through the
// simulator and checks net-out, fee, and remaining input against the bytecode-derived
// gross/unspent plus the venue's exact fee pipeline (floored fee, net rounds up).
func TestCalcAmountOutMatchesFixtures(t *testing.T) {
	sIn, vIn := fillableCases(t)
	for _, cases := range [][]fixtureCase{sIn, vIn} {
		for i, c := range cases {
			sim := simFromFixture(t, c, nil)
			res, err := calc(sim, c)
			require.NoError(t, err, "case %d", i)

			feeWad := uint256.NewInt(testFeeVIn)
			if c.stableIn {
				feeWad = uint256.NewInt(testFeeSIn)
			}
			var fee, netOut uint256.Int
			big256.MulDivDown(&fee, c.gross, feeWad, uWad)
			netOut.Sub(c.gross, &fee)

			require.Equal(t, netOut.Dec(), res.TokenAmountOut.Amount.String(), "case %d net", i)
			require.Equal(t, fee.Dec(), res.Fee.Amount.String(), "case %d fee", i)
			require.Equal(t, c.unspent.Dec(), res.RemainingTokenAmountIn.Amount.String(),
				"case %d remaining", i)

			si, ok := res.SwapInfo.(SwapInfo)
			require.True(t, ok)
			require.Equal(t, c.xAfter.Dec(), si.XAfter.Dec(), "case %d xAfter", i)
			require.Equal(t, c.gross.Dec(), si.GrossOut.Dec(), "case %d gross", i)
		}
	}
}

// TestSolvencyClamp: when the accounted output reserve is below the curve's gross quote,
// the payout is clamped to the reserve — exactly as CvammSwapLib.execute does.
func TestSolvencyClamp(t *testing.T) {
	sIn, _ := fillableCases(t)
	// stable in -> volatile out; pick a case with a sizeable gross and clamp the
	// volatile reserve below it.
	var c fixtureCase
	for _, cand := range sIn {
		if cand.gross.GtUint64(1_000_000) {
			c = cand
			break
		}
	}
	require.NotNil(t, c.gross, "no fixture with a sizeable gross")
	var clamped uint256.Int
	clamped.Rsh(c.gross, 1) // half the curve's gross
	sim := simFromFixture(t, c, entity.PoolReserves{hugeReserve, clamped.Dec()})
	res, err := calc(sim, c)
	require.NoError(t, err)

	var fee, netOut uint256.Int
	big256.MulDivDown(&fee, &clamped, uint256.NewInt(testFeeSIn), uWad)
	netOut.Sub(&clamped, &fee)
	require.Equal(t, netOut.Dec(), res.TokenAmountOut.Amount.String())

	// The zero-reserve book refuses the fill outright.
	sim = simFromFixture(t, c, entity.PoolReserves{hugeReserve, "0"})
	_, err = calc(sim, c)
	require.ErrorIs(t, err, ErrSwapExhausted)
}

// TestDustInputFullyUnspent: an input below normalized resolution (~kappa/WAD in the
// leg's own units) buys nothing and must be rejected, not quoted as a free fill.
func TestDustInputFullyUnspent(t *testing.T) {
	swaps, _ := loadFixtures(t)
	found := false
	for _, c := range swaps {
		if !c.unspent.Eq(c.amountIn) || c.amountIn.IsZero() {
			continue
		}
		found = true
		sim := simFromFixture(t, c, nil)
		_, err := calc(sim, c)
		require.ErrorIs(t, err, ErrSwapExhausted)
	}
	require.True(t, found, "fixtures must include fully-unspent dust cases")
}

func TestPausedAndRetracted(t *testing.T) {
	sIn, _ := fillableCases(t)
	c := sIn[0]

	sim := simFromFixture(t, c, nil)
	sim.Extra.Paused = true
	_, err := calc(sim, c)
	require.ErrorIs(t, err, ErrPaused)

	sim = simFromFixture(t, c, nil)
	sim.Extra.Kappa = uint256.NewInt(0)
	_, err = calc(sim, c)
	require.ErrorIs(t, err, ErrRetractedBook)
}

func TestExactOutRejected(t *testing.T) {
	sIn, _ := fillableCases(t)
	sim := simFromFixture(t, sIn[0], nil)
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{})
	require.ErrorIs(t, err, ErrExactOutNotSupported)
}

// TestUpdateBalanceAndClone: UpdateBalance replays the SwapInfo transition (input leg
// +used, output leg -gross, coordinate -> xAfter), quoting is deterministic, and a
// clone taken before the update is not affected by it.
func TestUpdateBalanceAndClone(t *testing.T) {
	sIn, _ := fillableCases(t)
	c := sIn[0]
	sim := simFromFixture(t, c, nil)

	res1, err := calc(sim, c)
	require.NoError(t, err)
	res2, err := calc(sim, c)
	require.NoError(t, err)
	require.Equal(t, res1.TokenAmountOut.Amount.String(), res2.TokenAmountOut.Amount.String(),
		"CalcAmountOut must be pure and deterministic")

	clone := sim.CloneState().(*PoolSimulator)

	si := res1.SwapInfo.(SwapInfo)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testStable, Amount: c.amountIn.ToBig()},
		TokenAmountOut: *res1.TokenAmountOut,
		Fee:            *res1.Fee,
		SwapInfo:       si,
	})

	require.Equal(t, si.XAfter.Dec(), sim.Extra.XWad.Dec())
	var wantStable uint256.Int
	base, _ := uint256.FromDecimal(hugeReserve)
	wantStable.Add(base, si.AmountInUsed)
	require.Equal(t, wantStable.Dec(), sim.reserveStable.Dec())
	var wantVol uint256.Int
	wantVol.Sub(base, si.GrossOut)
	require.Equal(t, wantVol.Dec(), sim.reserveVolatile.Dec())
	require.Equal(t, wantStable.ToBig().String(), sim.Info.Reserves[0].String())
	require.Equal(t, wantVol.ToBig().String(), sim.Info.Reserves[1].String())

	// The clone still prices from the pre-update state.
	require.Equal(t, c.x.Dec(), clone.Extra.XWad.Dec())
	resClone, err := calc(clone, c)
	require.NoError(t, err)
	require.Equal(t, res1.TokenAmountOut.Amount.String(), resClone.TokenAmountOut.Amount.String())

	// And updating the clone does not touch the (already updated) original.
	simX := sim.Extra.XWad.Dec()
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: resClone.SwapInfo.(SwapInfo)})
	require.Equal(t, simX, sim.Extra.XWad.Dec())
}

func TestMetaAndApproval(t *testing.T) {
	sIn, _ := fillableCases(t)
	sim := simFromFixture(t, sIn[0], nil)
	meta, ok := sim.GetMetaInfo(testStable, testVol).(PoolMeta)
	require.True(t, ok)
	require.Equal(t, testALM, meta.ALM)
	require.Equal(t, testALM, sim.GetApprovalAddress(testStable, testVol))
}
