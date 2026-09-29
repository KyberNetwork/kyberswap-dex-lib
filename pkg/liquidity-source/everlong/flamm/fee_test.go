package everlongflamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type feeGridRow struct {
	K      string    `json:"k"`
	I      string    `json:"i"`
	V      [8]string `json:"-"`
	RawV   any       `json:"v"`
	Val    string    `json:"-"`
	A      string    `json:"a"`
	B      string    `json:"b"`
	D      string    `json:"d"`
	P      string    `json:"p"`
	Kap    string    `json:"kap"`
	Band   string    `json:"band"`
	Rs     string    `json:"rs"`
	Rv     string    `json:"rv"`
	An     string    `json:"an"`
	Sp     string    `json:"sp"`
	Rvw    string    `json:"rvw"`
	LoanIn bool      `json:"loanIn"`
	F      string    `json:"f"`
	G      string    `json:"g"`
	W      string    `json:"w"`
	GOk    bool      `json:"gOk"`
	Ok     bool      `json:"ok"`
	Err    string    `json:"err"`
}

func feeParamsOf(t *testing.T, v [8]string) feeParams {
	return feeParams{
		MidFeeWad: *almU(t, v[0]), OutFeeWad: *almU(t, v[1]), GammaWad: *almU(t, v[2]), SigmaRefWad: *almU(t, v[3]),
		VolBetaWad: *almU(t, v[4]), VolMinWad: *almU(t, v[5]), VolMaxWad: *almU(t, v[6]), DirSkewWad: *almU(t, v[7]),
	}
}

// TestFeeFillGrid replays EverlongStrategy.fillFee (compiled from the c104 source behind a thin harness)
// over weights at, one wei around, and ramp-multiples away from the tie, both directions, curvature and
// volatility on and off, surcharge rows, one-sided books, dislocated spots and malformed rows.
func TestFeeFillGrid(t *testing.T) {
	t.Parallel()
	var rows []feeGridRow
	almLoadFixture(t, "fee_fill_grid.json.gz", &rows)
	params := map[string]feeParams{}
	n, lns, vols := 0, 0, 0
	for i := range rows {
		r := &rows[i]
		switch v := r.RawV.(type) {
		case string:
			r.Val = v
		case []any:
			for j := range r.V {
				r.V[j] = v[j].(string)
			}
		}
		switch r.K {
		case "params":
			params[r.I] = feeParamsOf(t, r.V)
			continue
		case "ln":
			v, err := feeLogRatioAbsWad(almU(t, r.A), almU(t, r.B))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Val, v.Dec(), "%+v", r)
			}
			lns++
			continue
		case "vol":
			p := params[r.P]
			v, err := feeVolMultiplier(&p, almU(t, r.Rvw), almU(t, r.D))
			if almExpectErr(t, r.Ok, r.Err, err, r) {
				require.Equal(t, r.Val, v.Dec(), "%+v", r)
			}
			vols++
			continue
		}
		p, ok := params[r.P]
		require.True(t, ok)
		s := feeState{ReserveStable: *almU(t, r.Rs), ReserveVolatile: *almU(t, r.Rv), AnchorWad: *almU(t, r.An),
			SpotWad: *almU(t, r.Sp), RvWad: *almU(t, r.Rvw)}
		f, err := feeFillFee(&p, &s, r.LoanIn, almU(t, r.Kap), almU(t, r.Band))
		if almExpectErr(t, r.Ok, r.Err, err, r) {
			require.Equal(t, r.F, f.Dec(), "%+v", r)
		}
		g, w, err := feeReductionG(&s, &p.GammaWad)
		if r.GOk {
			require.NoError(t, err, "%+v", r)
			require.Equal(t, r.G, g.Dec(), "%+v", r)
			require.Equal(t, r.W, w.Dec(), "%+v", r)
		} else {
			require.Error(t, err, "%+v", r)
		}
		n++
	}
	require.Greater(t, n, 6000)
	require.Equal(t, 600, lns)
	require.Equal(t, 300, vols)
}

// TestFeeLnWadEdges pins the two exact points of lnWad: ln(1) == 0 and the non-positive input revert.
func TestFeeLnWadEdges(t *testing.T) {
	t.Parallel()
	got, err := feeLnWad(uWad)
	require.NoError(t, err)
	require.True(t, got.IsZero())
	for _, x := range []*uint256.Int{uZero, new(uint256.Int).Lsh(uOne, 255), maxUint256} {
		_, err = feeLnWad(x)
		require.ErrorIs(t, err, errFeeLnWadUndefined)
	}
}

// ---------------------------------------------------------------------- DirSkewRamp.t.sol

// feeTestRow is the sealed e0_anchor fee row.
func feeTestRow() feeParams {
	return feeParams{
		MidFeeWad: *uint256.NewInt(0.030e18), OutFeeWad: *uint256.NewInt(0.005e18), GammaWad: *uint256.NewInt(0.05e18),
		SigmaRefWad: *uint256.NewInt(0.0004e18), VolBetaWad: *uint256.NewInt(4e18), VolMinWad: *uint256.NewInt(0.5e18),
		VolMaxWad: *uint256.NewInt(2e18), DirSkewWad: *uint256.NewInt(0.15e18),
	}
}

// feeTestState is DirSkewRampTest._state: a 1e24 book whose volatile value weight is wTarget, at the
// anchor WAD, with the given spot and rv 1e12.
func feeTestState(t *testing.T, wTarget, spot *uint256.Int) feeState {
	total := big256.TenPow(24)
	vv, err := almMulDiv(total, wTarget, uWad)
	require.NoError(t, err)
	var s feeState
	s.ReserveStable.Sub(total, &vv)
	s.ReserveVolatile = vv
	s.AnchorWad.Set(uWad)
	s.SpotWad.Set(spot)
	s.RvWad.SetUint64(1e12)
	return s
}

// feeTestOldLaw is DirSkewRampTest._old: the law as it shipped before the ramp, verbatim.
func feeTestOldLaw(t *testing.T, p *feeParams, s *feeState, loanAssetIn bool, kappa, band *uint256.Int) uint256.Int {
	g, w, err := feeReductionG(s, &p.GammaWad)
	require.NoError(t, err)
	if g.IsZero() {
		return p.OutFeeWad
	}
	disloc, err := feeLogRatioAbsWad(&s.SpotWad, &s.AnchorWad)
	require.NoError(t, err)
	v, err := feeVolMultiplier(p, &s.RvWad, &disloc)
	require.NoError(t, err)
	var span uint256.Int
	span.Sub(&p.MidFeeWad, &p.OutFeeWad)
	gv, _ := almMulDiv(&g, &v, uWad)
	f, _ := almMulDiv(&span, &gv, uWad)
	f.Add(&f, &p.OutFeeWad)
	if f.Gt(&p.MidFeeWad) {
		f.Set(&p.MidFeeWad)
	}
	restoring := s.SpotWad.Lt(&s.AnchorWad) == loanAssetIn
	var m uint256.Int
	if restoring {
		m.Sub(uWad, &p.DirSkewWad)
	} else {
		m.Add(uWad, &p.DirSkewWad)
	}
	increasing := w.Gt(almHalfWad)
	if loanAssetIn {
		increasing = w.Lt(almHalfWad)
	}
	if increasing && !kappa.IsZero() {
		var dev uint256.Int
		if w.Gt(almHalfWad) {
			dev.Sub(&w, almHalfWad)
		} else {
			dev.Sub(almHalfWad, &w)
		}
		if dev.Gt(band) {
			dev.Sub(&dev, band)
			add, _ := almMulDiv(kappa, &dev, uWad)
			m.Add(&m, &add)
		}
	}
	f, _ = almMulDiv(&f, &m, uWad)
	if f.Gt(uWad) {
		f.Set(uWad)
	}
	return f
}

func feeTestNew(t *testing.T, p *feeParams, s *feeState, loanAssetIn bool, kappa, band *uint256.Int) uint256.Int {
	f, err := feeFillFee(p, s, loanAssetIn, kappa, band)
	require.NoError(t, err)
	return f
}

func feeTestAbsDiff(a, b *uint256.Int) *uint256.Int {
	var d uint256.Int
	if a.Gt(b) {
		return d.Sub(a, b)
	}
	return d.Sub(b, a)
}

func feeTestW(half int64, dev uint64) *uint256.Int {
	var w uint256.Int
	if half >= 0 {
		return w.AddUint64(almHalfWad, dev)
	}
	return w.SubUint64(almHalfWad, dev)
}

// A: crossing the tie by a nano-weight no longer moves the quoted buy fee by more than dust.
func TestFeeDirSkewRampTieIsNoLongerAStep(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	kappa, band := uint256.NewInt(4e18), uint256.NewInt(0.06e18)
	var wadUp, wadDn uint256.Int
	wadUp.AddUint64(uWad, 1)
	wadDn.SubUint64(uWad, 1)
	lo := feeTestState(t, feeTestW(-1, 1e9), &wadUp)
	hi := feeTestState(t, feeTestW(1, 1e9), &wadDn)
	buyOld, buyOld2 := feeTestOldLaw(t, &p, &lo, true, kappa, band), feeTestOldLaw(t, &p, &hi, true, kappa, band)
	buyNew, buyNew2 := feeTestNew(t, &p, &lo, true, kappa, band), feeTestNew(t, &p, &hi, true, kappa, band)
	require.True(t, feeTestAbsDiff(&buyOld, &buyOld2).GtUint64(8e15), "the shipped law steps ~90 bp")
	require.True(t, feeTestAbsDiff(&buyNew, &buyNew2).LtUint64(1e13), "the ramped law does not step")
}

// B: from max(band, SKEW_RAMP_WAD) outward the ramped law is the shipped law to the wei.
func TestFeeDirSkewRampIdenticalOutsideTheRamp(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	kappa := uint256.NewInt(4e18)
	var wadUp, wadDn uint256.Int
	wadUp.AddUint64(uWad, 1)
	wadDn.SubUint64(uWad, 1)
	for _, b := range []uint64{0, 0.03e18, 0.06e18} {
		band := uint256.NewInt(b)
		ramp := max(b, 3e16)
		for i := uint64(1); i <= 40; i++ {
			dev := ramp + i*1e16/4
			if dev >= 5e17 {
				break
			}
			for side := 0; side < 2; side++ {
				var s feeState
				if side == 0 {
					s = feeTestState(t, feeTestW(-1, dev), &wadUp)
				} else {
					s = feeTestState(t, feeTestW(1, dev), &wadDn)
				}
				for _, loanIn := range []bool{true, false} {
					require.Equal(t, feeTestOldLaw(t, &p, &s, loanIn, kappa, band), feeTestNew(t, &p, &s, loanIn, kappa, band))
				}
			}
		}
	}
}

// C: calm two-sided flow pays what it paid: the two legs' sum is conserved to 2 wei inside the ramp.
func TestFeeDirSkewRampTwoSidedSumIsConserved(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	var wadDn uint256.Int
	wadDn.SubUint64(uWad, 1)
	for i := uint64(0); i <= 20; i++ {
		s := feeTestState(t, feeTestW(1, i*1e15), &wadDn)
		o1, o2 := feeTestOldLaw(t, &p, &s, true, uZero, uZero), feeTestOldLaw(t, &p, &s, false, uZero, uZero)
		n1, n2 := feeTestNew(t, &p, &s, true, uZero, uZero), feeTestNew(t, &p, &s, false, uZero, uZero)
		var sumOld, sumNew uint256.Int
		sumOld.Add(&o1, &o2)
		sumNew.Add(&n1, &n2)
		require.True(t, feeTestAbsDiff(&sumOld, &sumNew).LtUint64(3), "i=%d", i)
	}
}

// D: the widening leg is never cheaper, the envelope is unchanged and the quote has no step.
func TestFeeDirSkewRampContinuousAndBounded(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	kappa, band := uint256.NewInt(4e18), uint256.NewInt(0.06e18)
	var wadDn uint256.Int
	wadDn.SubUint64(uWad, 1)
	var prevWiden uint256.Int
	for i := uint64(0); i <= 200; i++ {
		dev := i * 1e15
		if dev >= 5e17 {
			break
		}
		s := feeTestState(t, feeTestW(1, dev), &wadDn)
		widen := feeTestNew(t, &p, &s, false, kappa, band)
		restore := feeTestNew(t, &p, &s, true, kappa, band)
		require.False(t, widen.Lt(&restore), "i=%d", i)
		// bare = old(0,0) * WAD / (WAD - dirSkew); widen <= bare * (WAD + dirSkew + kappa) / WAD + 1
		bareOld := feeTestOldLaw(t, &p, &s, true, uZero, uZero)
		var bare, env uint256.Int
		bare.Mul(&bareOld, uWad)
		bare.Div(&bare, env.Sub(uWad, &p.DirSkewWad))
		env.Add(uWad, &p.DirSkewWad)
		env.Add(&env, kappa)
		env.Mul(&bare, &env)
		env.Div(&env, uWad)
		env.AddUint64(&env, 1)
		require.False(t, widen.Gt(&env), "i=%d", i)
		if i > 0 {
			require.True(t, feeTestAbsDiff(&widen, &prevWiden).LtUint64(2e15), "i=%d", i)
		}
		prevWiden = widen
	}
}

// E: a one-sided book pays the bare out fee on both legs.
func TestFeeDirSkewRampOneSidedBookUnchanged(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	s := feeState{ReserveVolatile: *uint256.NewInt(1e18), AnchorWad: *uWad, SpotWad: *uWad}
	for _, loanIn := range []bool{true, false} {
		require.Equal(t, p.OutFeeWad, feeTestNew(t, &p, &s, loanIn, uint256.NewInt(4e18), uint256.NewInt(0.06e18)))
	}
}

// F: the peak fee the live row can quote is unchanged by the ramp.
func TestFeeDirSkewRampReachableMaximumUnchanged(t *testing.T) {
	t.Parallel()
	p := feeTestRow()
	kappa, band := uint256.NewInt(4e18), uint256.NewInt(0.06e18)
	var wadDn, maxOld, maxNew uint256.Int
	wadDn.SubUint64(uWad, 1)
	for i := uint64(1); i < 490; i++ {
		s := feeTestState(t, feeTestW(1, i*1e15), &wadDn)
		for _, loanIn := range []bool{true, false} {
			o, n := feeTestOldLaw(t, &p, &s, loanIn, kappa, band), feeTestNew(t, &p, &s, loanIn, kappa, band)
			if o.Gt(&maxOld) {
				maxOld = o
			}
			if n.Gt(&maxNew) {
				maxNew = n
			}
		}
	}
	require.Equal(t, maxOld, maxNew)
}
