package everlongflamm

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Wei-exact port of the fill-fee law in EverlongStrategy.sol (c104 @ 80abd43,
// src/hooks/everlong/EverlongStrategy.sol L132-198): reductionG, volMultiplier, logRatioAbsWad (Solady
// lnWad) and fillFee. The law carries no hot floor: the per-asset fee floor and cap are applied by
// the pool core, outside the hook. Validated against the c104 source over
// testdata/fee_fill_grid.json.gz.
//
// Relative to the audited Berachain fee law this is the c104 form:
//   - the directional skew is a continuous ramp over the book's own displacement |w - 1/2|, reaching
//     dirSkew at max(invSkewBand, SKEW_RAMP_WAD) instead of switching on spot < anchor;
//   - a zero reduction coefficient (zero curvature, or a one-sided book) returns outFee bare;
//   - the spot input is the hook's priceAtX(x) * reservationPrice / WAD, never a sqrt round trip.

// errFeeLnWadUndefined is Solady's LnWadUndefined(), raised when the ratio reads as a non-positive int256.
var errFeeLnWadUndefined = errors.New("everlong-flamm: LnWadUndefined()")

var (
	feeSkewRampWad = uint256.NewInt(3e16) // EverlongStrategy.SKEW_RAMP_WAD
	feeSigmaScale  = uint256.NewInt(1e9)
	feeFour        = uint256.NewInt(4)
)

// feeParams is EverlongStrategy.FeeParams (every field uint64 on-chain).
type feeParams struct {
	MidFeeWad   uint256.Int
	OutFeeWad   uint256.Int
	GammaWad    uint256.Int
	SigmaRefWad uint256.Int
	VolBetaWad  uint256.Int
	VolMinWad   uint256.Int
	VolMaxWad   uint256.Int
	DirSkewWad  uint256.Int
}

// feeState is EverlongStrategy.FeeState: the deployed legs (idle excluded), the reservation price, the
// live spot in the same frame and the realized-variance EMA.
type feeState struct {
	ReserveStable   uint256.Int
	ReserveVolatile uint256.Int
	AnchorWad       uint256.Int
	SpotWad         uint256.Int
	RvWad           uint256.Int
}

// feeReductionG is EverlongStrategy.reductionG: K = 4 Vs Vv / (Vs+Vv)^2 with total divided twice (so a
// small book cannot floor total^2 away), g = gK/(gK + WAD - K), and the volatile value weight. A
// one-sided book returns (0, 0). The total is summed before the one-sided test, as on-chain.
func feeReductionG(s *feeState, gammaWad *uint256.Int) (g, volatileWeight uint256.Int, err error) {
	vs := &s.ReserveStable
	vv, err := almMulDiv(&s.ReserveVolatile, &s.AnchorWad, uWad)
	if err != nil {
		return g, volatileWeight, err
	}
	var total, fourVs uint256.Int
	if _, overflow := total.AddOverflow(vs, &vv); overflow {
		return g, volatileWeight, errPanicArithmetic
	}
	if vs.IsZero() || vv.IsZero() {
		return g, volatileWeight, nil
	}
	if _, overflow := fourVs.MulOverflow(feeFour, vs); overflow {
		return g, volatileWeight, errPanicArithmetic
	}
	k, err := almMulDiv(&fourVs, uWad, &total)
	if err != nil {
		return g, volatileWeight, err
	}
	if k, err = almMulDiv(&k, &vv, &total); err != nil {
		return g, volatileWeight, err
	}
	gk, err := almMulDiv(gammaWad, &k, uWad)
	if err != nil {
		return g, volatileWeight, err
	}
	// gK + WAD - K, checked; zero exactly when gamma == 0 on a perfectly value-balanced book (K == WAD),
	// where the division panics.
	var denom uint256.Int
	if _, overflow := denom.AddOverflow(&gk, uWad); overflow {
		return g, volatileWeight, errPanicArithmetic
	}
	if denom.Lt(&k) {
		return g, volatileWeight, errPanicArithmetic
	}
	denom.Sub(&denom, &k)
	if g, err = almMulDiv(&gk, uWad, &denom); err != nil {
		return g, volatileWeight, err
	}
	volatileWeight, err = almMulDiv(&vv, uWad, &total)
	return g, volatileWeight, err
}

// feeVolMultiplier is EverlongStrategy.volMultiplier: clamp((sqrt(rv)*1e9/sigmaRef)(1 + beta*disloc),
// vMin, vMax); sigmaRef == 0 disables the term (WAD, unclamped).
func feeVolMultiplier(p *feeParams, rvWad, dislocWad *uint256.Int) (uint256.Int, error) {
	var v uint256.Int
	if p.SigmaRefWad.IsZero() {
		return *v.Set(uWad), nil
	}
	var sigma uint256.Int
	sigma.Sqrt(rvWad)
	sigma.Mul(&sigma, feeSigmaScale) // sqrt(2^256) * 1e9 < 2^158
	boost, err := almMulDiv(&p.VolBetaWad, dislocWad, uWad)
	if err != nil {
		return v, err
	}
	if _, overflow := boost.AddOverflow(&boost, uWad); overflow {
		return v, errPanicArithmetic
	}
	ratio, err := almMulDiv(&sigma, uWad, &p.SigmaRefWad)
	if err != nil {
		return v, err
	}
	if v, err = almMulDiv(&ratio, &boost, uWad); err != nil {
		return v, err
	}
	if v.Lt(&p.VolMinWad) {
		v.Set(&p.VolMinWad)
	} else if v.Gt(&p.VolMaxWad) {
		v.Set(&p.VolMaxWad)
	}
	return v, nil
}

// feeLogRatioAbsWad is EverlongStrategy.logRatioAbsWad: |ln(a/b)| in WAD, zero when either input is
// zero or the ratio floors to zero. The ratio is reinterpreted as int256 exactly as the cast does.
func feeLogRatioAbsWad(aWad, bWad *uint256.Int) (uint256.Int, error) {
	var r uint256.Int
	if aWad.IsZero() || bWad.IsZero() {
		return r, nil
	}
	ratio, err := almMulDiv(aWad, uWad, bWad)
	if err != nil || ratio.IsZero() {
		return r, err
	}
	if r, err = feeLnWad(&ratio); err != nil {
		return r, err
	}
	if r.Sign() < 0 {
		r.Neg(&r)
	}
	return r, nil
}

// Solady FixedPointMathLib.lnWad's rational-approximation constants.
var (
	feeLnP0 = big256.New("43456485725739037958740375743393")
	feeLnP1 = big256.New("24828157081833163892658089445524")
	feeLnP2 = big256.New("3273285459638523848632254066296")
	feeLnP3 = big256.New("11111509109440967052023855526967")
	feeLnP4 = big256.New("45023709667254063763336534515857")
	feeLnP5 = big256.New("14706773417378608786704636184526")
	feeLnP6 = new(uint256.Int).Lsh(big256.New("795164235651350426258249787498"), 96)
	feeLnQ0 = big256.New("5573035233440673466300451813936")
	feeLnQ1 = big256.New("71694874799317883764090561454958")
	feeLnQ2 = big256.New("283447036172924575727196451306956")
	feeLnQ3 = big256.New("401686690394027663651624208769553")
	feeLnQ4 = big256.New("204048457590392012362485061816622")
	feeLnQ5 = big256.New("31853899698501571402653359427138")
	feeLnQ6 = big256.New("909429971244387300277376558375")
	feeLnS  = big256.New("1677202110996718588342820967067443963516166")
	feeLnK  = big256.New("16597577552685614221487285958193947469193820559219878177908093499208371")
	feeLnC  = big256.New("600920179829731861736702779321621459595472258049074101567377883020018308")
)

// feeLnWad is Solady FixedPointMathLib.lnWad, opcode for opcode: x and the result are int256 in two's
// complement, and every add/sub/mul wraps mod 2^256 while sar and sdiv are signed, as in the EVM.
func feeLnWad(x *uint256.Int) (uint256.Int, error) {
	var p uint256.Int
	if x.Sign() <= 0 {
		return p, errFeeLnWadUndefined
	}
	// r = 255 ^ log2(x) = 256 - bitlen(x) for a positive int256.
	r := uint64(256 - x.BitLen())
	var xn, q, t uint256.Int
	xn.Lsh(x, uint(r))
	xn.Rsh(&xn, 159)

	// p = sar(96, (P0 + sar(96, (P1 + sar(96, (P2 + x) * x)) * x)) * x) - P3
	t.Add(feeLnP2, &xn)
	t.Mul(&t, &xn)
	t.SRsh(&t, 96)
	t.Add(feeLnP1, &t)
	t.Mul(&t, &xn)
	t.SRsh(&t, 96)
	t.Add(feeLnP0, &t)
	t.Mul(&t, &xn)
	p.SRsh(&t, 96)
	p.Sub(&p, feeLnP3)
	p.Mul(&p, &xn)
	p.SRsh(&p, 96)
	p.Sub(&p, feeLnP4)
	p.Mul(&p, &xn)
	p.SRsh(&p, 96)
	p.Sub(&p, feeLnP5)
	p.Mul(&p, &xn)
	p.Sub(&p, feeLnP6)

	q.Add(feeLnQ0, &xn)
	for _, c := range [...]*uint256.Int{feeLnQ1, feeLnQ2, feeLnQ3, feeLnQ4, feeLnQ5, feeLnQ6} {
		q.Mul(&xn, &q)
		q.SRsh(&q, 96)
		q.Add(c, &q)
	}

	p.SDiv(&p, &q)
	p.Mul(feeLnS, &p)
	var k uint256.Int
	k.SetUint64(159)
	k.Sub(&k, t.SetUint64(r))
	k.Mul(feeLnK, &k)
	p.Add(&k, &p)
	p.Add(feeLnC, &p)
	p.SRsh(&p, 174)
	return p, nil
}

// feeFillFee is EverlongStrategy.fillFee: min(out + (mid - out) g v, mid) times the directional
// multiplier (1 -/+ skew(|w - 1/2|)) + invSkewKappa * max(0, |w - 1/2| - band) (the surcharge only on
// fills that increase the displacement), clamped at 100%. skew is proportional to the displacement up
// to max(band, SKEW_RAMP_WAD) and dirSkew beyond, so both legs quote the same fee at the tie. A zero g
// (zero curvature or a one-sided book) returns outFee bare.
// loanAssetIn is true when the taker pays the stable leg (a buy of the volatile asset).
func feeFillFee(p *feeParams, s *feeState, loanAssetIn bool, invSkewKappaWad, invSkewBandWad *uint256.Int) (uint256.Int,
	error) {
	var f uint256.Int
	g, w, err := feeReductionG(s, &p.GammaWad)
	if err != nil {
		return f, err
	}
	if g.IsZero() {
		return *f.Set(&p.OutFeeWad), nil
	}
	disloc, err := feeLogRatioAbsWad(&s.SpotWad, &s.AnchorWad)
	if err != nil {
		return f, err
	}
	v, err := feeVolMultiplier(p, &s.RvWad, &disloc)
	if err != nil {
		return f, err
	}
	if p.MidFeeWad.Lt(&p.OutFeeWad) { // uint64 checked subtraction
		return f, errPanicArithmetic
	}
	var span uint256.Int
	span.Sub(&p.MidFeeWad, &p.OutFeeWad)
	gv, err := almMulDiv(&g, &v, uWad)
	if err != nil {
		return f, err
	}
	if f, err = almMulDiv(&span, &gv, uWad); err != nil {
		return f, err
	}
	if _, overflow := f.AddOverflow(&p.OutFeeWad, &f); overflow {
		return f, errPanicArithmetic
	}
	if f.Gt(&p.MidFeeWad) {
		f.Set(&p.MidFeeWad)
	}

	// Which way the fill pushes the book, read once off the value weight: paying stable removes
	// volatile, so a buy increases the displacement only below one half.
	increasing := w.Gt(almHalfWad)
	if loanAssetIn {
		increasing = w.Lt(almHalfWad)
	}
	var dev, ramp, skew, multiplier uint256.Int
	if w.Gt(almHalfWad) {
		dev.Sub(&w, almHalfWad)
	} else {
		dev.Sub(almHalfWad, &w)
	}
	ramp.Set(feeSkewRampWad)
	if invSkewBandWad.Gt(feeSkewRampWad) {
		ramp.Set(invSkewBandWad)
	}
	if !dev.Lt(&ramp) {
		skew.Set(&p.DirSkewWad)
	} else if skew, err = almMulDiv(&p.DirSkewWad, &dev, &ramp); err != nil {
		return f, err
	}
	if increasing {
		multiplier.Add(uWad, &skew)
	} else {
		if uWad.Lt(&skew) {
			return f, errPanicArithmetic
		}
		multiplier.Sub(uWad, &skew)
	}
	if increasing && !invSkewKappaWad.IsZero() && dev.Gt(invSkewBandWad) {
		var excess uint256.Int
		excess.Sub(&dev, invSkewBandWad)
		surcharge, err := almMulDiv(invSkewKappaWad, &excess, uWad)
		if err != nil {
			return f, err
		}
		if _, overflow := multiplier.AddOverflow(&multiplier, &surcharge); overflow {
			return f, errPanicArithmetic
		}
	}
	if f, err = almMulDiv(&f, &multiplier, uWad); err != nil {
		return f, err
	}
	if f.Gt(uWad) {
		f.Set(uWad)
	}
	return f, nil
}
