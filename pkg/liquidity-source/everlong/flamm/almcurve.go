package everlongflamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Wei-exact port of the swap/book half of AlmCurve.sol (c104 @ 80abd43,
// src/hooks/everlong/AlmCurve.sol): the normalized reservation curve the EverlongHook trades on.
// Normalized inventory (x volatile, y stable) sits on y^2 + (x+c)y - k/x = 0 with c = 1/(2A) - 1 and
// 4k = 1/(2A); the hook holds a band of it (almSupport) scaled by kappa and located by an anchor sqrt
// price. The recenter-only solvers (reseed, reseedAt, xAtValueRatio, _scaleAt) are not on the swap
// path and are not ported. Validated against the deployed Base library over
// testdata/alm_curve_grid.json.gz.
//
// Every division floors unless named Ceil and sqrt is floor-sqrt, exactly as on-chain. Two properties
// are load-bearing at the wei scale and must survive any edit:
//   - almYAtX forms the radicand as ONE sum before the square root. Two independently floored terms
//     move in opposite directions in x, so their sum can step UP as x increases — a non-monotone y,
//     which the stable-in bisection brackets on.
//   - c is negative for every valid A (A > WAD/2), so b = x + c is carried as a magnitude plus sign
//     to avoid the catastrophic cancellation of the naive (-b + root)/2 form.

var (
	almMinXWad = uint256.NewInt(1e15) // AlmCurve.MIN_X_WAD
	almMaxXWad = uint256.NewInt(1999e15)
	almMinAWad = uint256.NewInt(5e17 + 1)
	almMaxAWad = new(uint256.Int).Set(big256.TenPow(21)) // 1000e18
	almHalfWad = uint256.NewInt(5e17)
)

// almSeedWindow is the conservative bracket around the x<->y symmetry seed (AlmCurve._seedBracket).
const almSeedWindow = 1 << 16

// almSupport is AlmCurve.Support: the funded band [xLo, xHi] of the curve for amplification AWad, with
// YHi = y(xHi) the stable offset.
type almSupport struct {
	AWad uint256.Int
	XLo  uint256.Int
	XHi  uint256.Int
	YHi  uint256.Int
}

// almMulDiv is OpenZeppelin 4.x Math.mulDiv(x, y, d): floor(x*y/d) over a 512-bit product, reverting
// exactly where it does.
func almMulDiv(x, y, d *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if d.IsZero() {
		if _, overflow := z.MulOverflow(x, y); overflow {
			return uint256.Int{}, errMulDivOverflow
		}
		return uint256.Int{}, errPanicDivZero
	}
	if _, overflow := z.MulDivOverflow(x, y, d); overflow {
		return uint256.Int{}, errMulDivOverflow
	}
	return z, nil
}

// almMulDivUp is Math.mulDiv(x, y, d, Rounding.Up): the floor plus one on a non-zero remainder, the
// increment checked.
func almMulDivUp(x, y, d *uint256.Int) (uint256.Int, error) {
	z, err := almMulDiv(x, y, d)
	if err != nil {
		return z, err
	}
	var rem uint256.Int
	if !rem.MulMod(x, y, d).IsZero() {
		if z.Eq(maxUint256) {
			return uint256.Int{}, errPanicArithmetic
		}
		z.AddUint64(&z, 1)
	}
	return z, nil
}

// almNegC is -cWad(aWad) = WAD - WAD^2/(2A), the magnitude of the always-negative c, together with
// fourK = WAD^2/(2A). AlmCurve.cWad reverts CurveAmplification outside [MIN_A_WAD, MAX_A_WAD].
func almNegC(aWad *uint256.Int) (negC, fourK uint256.Int, err error) {
	if aWad.Lt(almMinAWad) || aWad.Gt(almMaxAWad) {
		return negC, fourK, ErrCurveAmplification
	}
	var twoA uint256.Int
	twoA.Lsh(aWad, 1)
	fourK.Div(uWadSquared, &twoA)
	negC.Sub(uWad, &fourK)
	return negC, fourK, nil
}

// almYAtX is AlmCurve.yAtX: the positive root of y^2 + (x+c)y - k/x = 0, as (root -/+ |b|)/2 branching
// on sign(b). The domain check precedes the amplification check, as on-chain.
func almYAtX(x, aWad *uint256.Int) (uint256.Int, error) {
	var y uint256.Int
	if x.Lt(almMinXWad) || x.Gt(almMaxXWad) {
		return y, ErrCurveDomain
	}
	negC, fourK, err := almNegC(aWad)
	if err != nil {
		return y, err
	}
	// b = x + c = x - |c|
	var absB uint256.Int
	bNeg := x.Lt(&negC)
	if bNeg {
		absB.Sub(&negC, x)
	} else {
		absB.Sub(x, &negC)
	}
	// ONE floor in the radicand: absB^2 is exact (<= 4e36), the k-term floors once (<= 1e39).
	var rad, term uint256.Int
	big256.MulDivDown(&term, &fourK, uWadSquared, x)
	rad.Mul(&absB, &absB)
	rad.Add(&rad, &term)
	y.Sqrt(&rad)
	if bNeg {
		y.Add(&y, &absB)
	} else {
		y.Sub(&y, &absB)
	}
	y.Rsh(&y, 1)
	return y, nil
}

// almPriceAtX is AlmCurve.priceAtX: the normalized marginal price F_x/F_y = y(2x+y+c) / x(x+2y+c),
// exactly WAD at x = WAD/2. A non-positive numerator or denominator reverts CurveDomain.
func almPriceAtX(x, aWad *uint256.Int) (uint256.Int, error) {
	y, err := almYAtX(x, aWad)
	if err != nil {
		return y, err
	}
	negC, _, err := almNegC(aWad)
	if err != nil {
		return y, err
	}
	// num = y * (2x + y + c), den = x * (x + 2y + c); every term is < 2^70, so the signed sums are
	// formed as unsigned sums compared against |c|.
	var sNum, sDen, num, den, p uint256.Int
	sNum.Lsh(x, 1)
	sNum.Add(&sNum, &y)
	sDen.Lsh(&y, 1)
	sDen.Add(&sDen, x)
	if !sNum.Gt(&negC) || !sDen.Gt(&negC) || y.IsZero() {
		return p, ErrCurveDomain
	}
	sNum.Sub(&sNum, &negC)
	sDen.Sub(&sDen, &negC)
	num.Mul(&y, &sNum)
	den.Mul(x, &sDen)
	big256.MulDivDown(&p, &num, uWad, &den)
	return p, nil
}

// almXAtPrice is AlmCurve.xAtPrice: the coordinate whose marginal price is targetPriceWad, by
// bisection over the whole domain (clamping at the edges). Used only by almSupportFor.
func almXAtPrice(targetPriceWad, aWad *uint256.Int) (uint256.Int, error) {
	var lo, hi uint256.Int
	lo.Set(almMinXWad)
	hi.Set(almMaxXWad)
	pLo, err := almPriceAtX(&lo, aWad)
	if err != nil {
		return lo, err
	}
	if !targetPriceWad.Lt(&pLo) {
		return lo, nil
	}
	pHi, err := almPriceAtX(&hi, aWad)
	if err != nil {
		return lo, err
	}
	if !targetPriceWad.Gt(&pHi) {
		return hi, nil
	}
	var mid uint256.Int
	for i := 0; i < 128; i++ {
		mid.Add(&lo, &hi)
		mid.Rsh(&mid, 1)
		if mid.Eq(&lo) {
			break
		}
		pMid, err := almPriceAtX(&mid, aWad)
		if err != nil {
			return mid, err
		}
		if pMid.Gt(targetPriceWad) {
			lo.Set(&mid)
		} else {
			hi.Set(&mid)
		}
	}
	mid.Add(&lo, &hi)
	mid.Rsh(&mid, 1)
	return mid, nil
}

// almSupportFor is AlmCurve.supportFor(aWad, spanUpWad, spanDnWad): the funded band from a price span,
// rejecting a span the domain clamp would silently widen.
func almSupportFor(aWad, spanUpWad, spanDnWad *uint256.Int) (almSupport, error) {
	var sup almSupport
	if !spanUpWad.Gt(uWad) || !spanDnWad.Gt(uWad) {
		return sup, ErrCurveSpan
	}
	xLo, err := almXAtPrice(spanUpWad, aWad)
	if err != nil {
		return sup, err
	}
	var loPrice uint256.Int
	big256.MulDivDown(&loPrice, uWad, uWad, spanDnWad)
	xHi, err := almXAtPrice(&loPrice, aWad)
	if err != nil {
		return sup, err
	}
	if !xLo.Gt(almMinXWad) || !xHi.Lt(almMaxXWad) || !xLo.Lt(&xHi) {
		return sup, ErrCurveSpan
	}
	yHi, err := almYAtX(&xHi, aWad)
	if err != nil {
		return sup, err
	}
	sup.AWad.Set(aWad)
	sup.XLo, sup.XHi, sup.YHi = xLo, xHi, yHi
	return sup, nil
}

// almHeldAt is AlmCurve.heldAt: the normalized inventory held at xWad, clamped to the band — volatile
// x - xLo and stable y(x) - yHi (zero-guarded).
func almHeldAt(sup *almSupport, xWad *uint256.Int) (volatileWad, stableWad uint256.Int, err error) {
	var x uint256.Int
	switch {
	case xWad.Lt(&sup.XLo):
		x.Set(&sup.XLo)
	case xWad.Gt(&sup.XHi):
		x.Set(&sup.XHi)
	default:
		x.Set(xWad)
	}
	if x.Lt(&sup.XLo) { // only a malformed band (xHi < xLo) reaches this checked subtraction
		return volatileWad, stableWad, errPanicArithmetic
	}
	volatileWad.Sub(&x, &sup.XLo)
	y, err := almYAtX(&x, &sup.AWad)
	if err != nil {
		return volatileWad, stableWad, err
	}
	if y.Gt(&sup.YHi) {
		stableWad.Sub(&y, &sup.YHi)
	}
	return volatileWad, stableWad, nil
}

// almReservesAt is AlmCurve.reservesAt: token reserves at xWad for anchor and scale,
// stable = kappa*held*a/Q96 and volatile = kappa*held*Q96/a, both legs floored twice.
func almReservesAt(sup *almSupport, anchorSqrtX96, kappa, xWad *uint256.Int) (stable, volatileAmount uint256.Int,
	err error) {
	if anchorSqrtX96.IsZero() {
		return stable, volatileAmount, ErrCurveDomain
	}
	volatileHeld, stableHeld, err := almHeldAt(sup, xWad)
	if err != nil {
		return stable, volatileAmount, err
	}
	t, err := almMulDiv(kappa, &stableHeld, uWad)
	if err != nil {
		return stable, volatileAmount, err
	}
	if stable, err = almMulDiv(&t, anchorSqrtX96, uQ96); err != nil {
		return stable, volatileAmount, err
	}
	if t, err = almMulDiv(kappa, &volatileHeld, uWad); err != nil {
		return stable, volatileAmount, err
	}
	volatileAmount, err = almMulDiv(&t, uQ96, anchorSqrtX96)
	return stable, volatileAmount, err
}

// almSwapExactIn is AlmCurve.swapExactIn: an exact-input fill in normalized units. Volatile-in (x rises)
// is closed form; stable-in inverts y -> x with the seeded bisection. inputUnused is non-zero only where
// the funded band truncates the fill.
func almSwapExactIn(sup *almSupport, xWad *uint256.Int, volatileIn bool, amountInWad *uint256.Int) (amountOut,
	xAfter, inputUnused uint256.Int, err error) {
	// A no-input fill MUST be a no-op: the stable-in bisection lands on the leftmost coordinate of a
	// floored-y tread, so without this guard a zero input could move the coordinate.
	if amountInWad.IsZero() {
		xAfter.Set(xWad)
		return amountOut, xAfter, inputUnused, nil
	}
	y, err := almYAtX(xWad, &sup.AWad)
	if err != nil {
		return amountOut, xAfter, inputUnused, err
	}
	if !volatileIn {
		return almStableIn(sup, xWad, &y, amountInWad)
	}
	var room, used uint256.Int
	if sup.XHi.Gt(xWad) {
		room.Sub(&sup.XHi, xWad)
	}
	used.Set(amountInWad)
	if used.Gt(&room) {
		used.Set(&room)
	}
	inputUnused.Sub(amountInWad, &used)
	xAfter.Add(xWad, &used)
	yAfter, err := almYAtX(&xAfter, &sup.AWad)
	if err != nil {
		return amountOut, xAfter, inputUnused, err
	}
	if y.Gt(&yAfter) {
		amountOut.Sub(&y, &yAfter)
	}
	return amountOut, xAfter, inputUnused, nil
}

// almStableIn is AlmCurve._stableIn: paying the stable leg raises y and lowers x; the root is the
// rightmost x whose floored y exceeds y + used (the predicate flips true -> false at hi).
func almStableIn(sup *almSupport, xWad, y, amountInWad *uint256.Int) (amountOut, xAfter, inputUnused uint256.Int,
	err error) {
	yMax, err := almYAtX(&sup.XLo, &sup.AWad)
	if err != nil {
		return amountOut, xAfter, inputUnused, err
	}
	var reachable, used uint256.Int
	if yMax.Gt(y) {
		reachable.Sub(&yMax, y)
	}
	used.Set(amountInWad)
	if used.Gt(&reachable) {
		used.Set(&reachable)
	}
	inputUnused.Sub(amountInWad, &used)
	if used.IsZero() {
		xAfter.Set(xWad)
		inputUnused.Set(amountInWad)
		return amountOut, xAfter, inputUnused, nil
	}
	var lo, hi, yTarget uint256.Int
	lo.Set(&sup.XLo)
	hi.Set(xWad)
	yTarget.Add(y, &used)
	if err = almSeedBracket(sup, &lo, &hi, &yTarget); err != nil {
		return amountOut, xAfter, inputUnused, err
	}
	var mid uint256.Int
	for i := 0; i < 128; i++ {
		mid.Add(&lo, &hi)
		mid.Rsh(&mid, 1)
		if mid.Eq(&lo) {
			break
		}
		yMid, err := almYAtX(&mid, &sup.AWad)
		if err != nil {
			return amountOut, xAfter, inputUnused, err
		}
		if yMid.Gt(&yTarget) {
			lo.Set(&mid)
		} else {
			hi.Set(&mid)
		}
	}
	xAfter.Set(&hi)
	if xWad.Gt(&xAfter) {
		amountOut.Sub(xWad, &xAfter)
	}
	return amountOut, xAfter, inputUnused, nil
}

// almSeedBracket is AlmCurve._seedBracket: narrow [lo, hi] around yAtX(yTarget) — valid because the
// level set is symmetric under swapping the coordinates — and adopt the window ONLY if it brackets
// the root under the bisection's own predicate. It never changes the bisection's answer.
func almSeedBracket(sup *almSupport, lo, hi, yTarget *uint256.Int) error {
	if yTarget.Lt(almMinXWad) || yTarget.Gt(almMaxXWad) {
		return nil
	}
	seed, err := almYAtX(yTarget, &sup.AWad)
	if err != nil {
		return err
	}
	var loSeed, hiSeed, tmp uint256.Int
	if tmp.AddUint64(lo, almSeedWindow); seed.Gt(&tmp) {
		loSeed.SubUint64(&seed, almSeedWindow)
	} else {
		loSeed.Set(lo)
	}
	if tmp.AddUint64(&seed, almSeedWindow); tmp.Lt(hi) {
		hiSeed.Set(&tmp)
	} else {
		hiSeed.Set(hi)
	}
	if !loSeed.Lt(&hiSeed) {
		return nil
	}
	yEdge, err := almYAtX(&loSeed, &sup.AWad)
	if err != nil {
		return err
	}
	if !yEdge.Gt(yTarget) {
		return nil
	}
	if yEdge, err = almYAtX(&hiSeed, &sup.AWad); err != nil {
		return err
	}
	if yEdge.Gt(yTarget) {
		return nil
	}
	lo.Set(&loSeed)
	hi.Set(&hiSeed)
	return nil
}

// almToNormalized is AlmCurve._toNormalized: token amount to normalized inventory, FLOORING — the
// inverse of almToToken's two mulDivs, applied in the same order.
func almToNormalized(amount, anchorSqrtX96, kappa *uint256.Int, stable bool) (uint256.Int, error) {
	t, err := almMulDiv(amount, uWad, kappa)
	if err != nil {
		return t, err
	}
	if stable {
		return almMulDiv(&t, uQ96, anchorSqrtX96)
	}
	return almMulDiv(&t, anchorSqrtX96, uQ96)
}

// almToToken is AlmCurve._toToken: normalized inventory to a token amount, FLOORING; mirrors
// almReservesAt, one anchor factor per leg.
func almToToken(norm, anchorSqrtX96, kappa *uint256.Int, stable bool) (uint256.Int, error) {
	t, err := almMulDiv(kappa, norm, uWad)
	if err != nil {
		return t, err
	}
	if stable {
		return almMulDiv(&t, anchorSqrtX96, uQ96)
	}
	return almMulDiv(&t, uQ96, anchorSqrtX96)
}

// almToTokenCeil is AlmCurve._toTokenCeil: almToToken rounding UP at both steps, used only to price a
// truncated fill's charge.
func almToTokenCeil(norm, anchorSqrtX96, kappa *uint256.Int, stable bool) (uint256.Int, error) {
	t, err := almMulDivUp(kappa, norm, uWad)
	if err != nil {
		return t, err
	}
	if stable {
		return almMulDivUp(&t, anchorSqrtX96, uQ96)
	}
	return almMulDivUp(&t, uQ96, anchorSqrtX96)
}

// almSwapExactInX96 is AlmCurve.swapExactInX96: the exact-input fill in TOKEN units, amountOut gross of
// any fee. An input below normalized resolution (~kappa/WAD in the leg's units) is reported entirely
// unspent; an untruncated fill charges amountIn exactly; a fill truncated by the band charges the
// used movement rounded UP, capped at amountIn.
func almSwapExactInX96(sup *almSupport, anchorSqrtX96, kappa, xWad *uint256.Int, stableIn bool,
	amountIn *uint256.Int) (amountOut, xAfter, amountInUnspent uint256.Int, err error) {
	if anchorSqrtX96.IsZero() || kappa.IsZero() {
		return amountOut, xAfter, amountInUnspent, ErrCurveDomain
	}
	inNorm, err := almToNormalized(amountIn, anchorSqrtX96, kappa, stableIn)
	if err != nil {
		return amountOut, xAfter, amountInUnspent, err
	}
	if inNorm.IsZero() {
		xAfter.Set(xWad)
		amountInUnspent.Set(amountIn)
		return amountOut, xAfter, amountInUnspent, nil
	}
	// The curve's flag is VOLATILE-in; this one is STABLE-in — opposites, as are the output legs.
	outNorm, xAfter, unusedNorm, err := almSwapExactIn(sup, xWad, !stableIn, &inNorm)
	if err != nil {
		return amountOut, xAfter, amountInUnspent, err
	}
	if amountOut, err = almToToken(&outNorm, anchorSqrtX96, kappa, !stableIn); err != nil {
		return amountOut, xAfter, amountInUnspent, err
	}
	if unusedNorm.IsZero() {
		return amountOut, xAfter, amountInUnspent, nil
	}
	var usedNorm uint256.Int
	usedNorm.Sub(&inNorm, &unusedNorm)
	usedTok, err := almToTokenCeil(&usedNorm, anchorSqrtX96, kappa, stableIn)
	if err != nil {
		return amountOut, xAfter, amountInUnspent, err
	}
	if amountIn.Gt(&usedTok) {
		amountInUnspent.Sub(amountIn, &usedTok)
	}
	return amountOut, xAfter, amountInUnspent, nil
}
