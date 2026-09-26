package everlongflamm

import (
	"math/bits"

	"github.com/holiman/uint256"
)

// Port of the frozen c1 leverage curve, CollRebalancerMath (src/hooks/everlong/lev/CollRebalancerMath.sol
// @ c104 80abd43, linked on Base at 0xC002d0731E6a2E6e80Be754779bCEf6B01Aff0bb) and its LevCurveTypes tuple.
// Every quantity is a uint256 and every step keeps Solidity's operation order: `/` and Math.mulDiv floor,
// Math.Rounding.Up ceils, Math.sqrt floors and Mul512.productGt compares the two 512-bit products exactly.
//
// The library never reverts. Its entrypoints bound cv and debt by MAX_INPUT = 1e38 before any raw product
// forms, and the widest of those products (sum*sum in _rootIntervalContains and _cvRequiredOnAnchor) peaks
// at ~7.1e76 < 2^256, so the wrapping uint256 arithmetic below is exact wherever Solidity's checked
// arithmetic would have been, and the overflow flags of the shared mulDiv helpers are ignored here on the
// same bound.
//
// Every returned word is freshly allocated: no result aliases an argument or a package constant, so callers
// may mutate what they get back (TestAliasingReturnedWords).

var (
	levMaxInput = uint256.MustFromDecimal("100000000000000000000000000000000000000") // MAX_INPUT = 1e38

	// levLeverageRatioWad is LEVERAGE_RATIO_WAD = floor(4e18/9), the only rWad the curve accepts.
	levLeverageRatioWad = uint256.NewInt(444_444_444_444_444_444)

	// Floors of the frozen 155%-wall curve in normalized h=cv/T and D=d/T space (frozenParams().curve).
	levHZero = uint256.NewInt(562_500_000_000_000_000) // 9/16
	levHJoin = uint256.NewInt(1_010_000_000_000_000_000)
	levHWall = uint256.NewInt(1_882_448_291_726_770_582)
	levWidth = uint256.NewInt(872_448_291_726_770_582)
	levDJoin = uint256.NewInt(509_975_124_224_178_054)
	levDWall = uint256.NewInt(1_214_482_768_855_981_020)

	// Cubic Bezier controls for phi=dD/dh on the transition.
	levP0 = uint256.NewInt(995_037_190_209_989_135)
	levP1 = uint256.NewInt(851_783_312_849_706_840)
	levP2 = uint256.NewInt(738_044_106_433_170_508)
	levP3 = uint256.NewInt(645_161_290_322_580_645)

	// Quartic Bezier controls for the exact integral of the cubic above (Q0 = 0).
	levQ1 = uint256.NewInt(248_759_297_552_497_283)
	levQ2 = uint256.NewInt(461_705_125_764_923_993)
	levQ3 = uint256.NewInt(646_216_152_373_216_620)
	levQ4 = uint256.NewInt(807_506_474_953_861_782)

	// levTargetSpreadCapPpm is AT_OR_BELOW_TARGET_SPREAD_CAP_PPM: the deleverage spread ceiling on collateral
	// released while the position is at or below its 200% target.
	levTargetSpreadCapPpm = uint256.NewInt(13_000)
	// levDustAnchorFloor is _deleverageProRata's dust guard: anchors below it are charged the posted rate.
	levDustAnchorFloor = uint256.NewInt(101)

	levHalfLawOffset = uint256.NewInt(1_500_000_000_000_000_000) // _debtNormWad's 1.5 WAD
	levThreeWad      = uint256.NewInt(3_000_000_000_000_000_000)
	levWadMinusOne   = uint256.NewInt(999_999_999_999_999_999)
	levTwo           = uint256.NewInt(2)
	levThree         = uint256.NewInt(3)
	levSix           = uint256.NewInt(6)
	levEight         = uint256.NewInt(8)

	// levRhoWallNumerator is D_WALL*WAD^2 and levRhoDenominator H_WALL*WAD^2 (_recoveryDebtAtY).
	levRhoWallNumerator = new(uint256.Int).Mul(levDWall, uWadSquared)
	levRhoDenominator   = new(uint256.Int).Mul(levHWall, uWadSquared)
	levHWallMinusDWall  = new(uint256.Int).Sub(levHWall, levDWall)
)

// levMul512 returns the high and low limbs of the full 512-bit product a*b (Mul512.mul512).
func levMul512(a, b *uint256.Int) (hi, lo uint256.Int) {
	var p [8]uint64
	for i := 0; i < 4; i++ {
		var carry uint64
		for j := 0; j < 4; j++ {
			h, l := bits.Mul64(a[i], b[j])
			var c uint64
			l, c = bits.Add64(l, p[i+j], 0)
			h += c
			l, c = bits.Add64(l, carry, 0)
			h += c
			p[i+j] = l
			carry = h
		}
		p[i+4] = carry
	}
	lo = uint256.Int{p[0], p[1], p[2], p[3]}
	hi = uint256.Int{p[4], p[5], p[6], p[7]}
	return hi, lo
}

// levProductGt is Mul512.productGt: a*b > c*d over the full 512-bit products.
func levProductGt(a, b, c, d *uint256.Int) bool {
	hi1, lo1 := levMul512(a, b)
	hi2, lo2 := levMul512(c, d)
	return hi1.Gt(&hi2) || (hi1.Eq(&hi2) && lo1.Gt(&lo2))
}

func levMul(x, y *uint256.Int) *uint256.Int { return new(uint256.Int).Mul(x, y) }
func levAdd(x, y *uint256.Int) *uint256.Int { return new(uint256.Int).Add(x, y) }
func levSub(x, y *uint256.Int) *uint256.Int { return new(uint256.Int).Sub(x, y) }

// levMulDivFloor and levMulDivUp are Math.mulDiv on inputs the curve has already bounded (see the file
// header), so the overflow flag cannot be set.
func levMulDivFloor(x, y, d *uint256.Int) *uint256.Int {
	z, _ := mulDiv(x, y, d)
	return z
}

func levMulDivUp(x, y, d *uint256.Int) *uint256.Int {
	z, _ := mulDivCeil(x, y, d)
	return z
}

// levCurveFrozenParams mirrors CollRebalancerMath.frozenParams() field for field (LevCurveParams): the curve
// tuple, hZero, leverageRatioWad and targetSpreadCapPpm. The library leaves the four caller-owned fields zero.
// The words are copies, so a caller cannot write through them into the curve constants.
func levCurveFrozenParams() [16]*uint256.Int {
	params := [16]*uint256.Int{
		levHJoin, levHWall, levWidth, levDJoin, levDWall, levP0, levP1, levP2, levP3, levQ1, levQ2, levQ3, levQ4,
		levHZero, levLeverageRatioWad, levTargetSpreadCapPpm,
	}
	for i, w := range params {
		params[i] = new(uint256.Int).Set(w)
	}
	return params
}

// levRefused is the (0, collateral, debt) refusal triple of deleverageQuote and leverageQuote, as fresh words:
// a caller mutating the result must reach neither its own inputs nor a package constant.
func levRefused(collateral, debt *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	return new(uint256.Int), new(uint256.Int).Set(collateral), new(uint256.Int).Set(debt)
}

// levMarkedValue is _markedValue: cv = floor(collateral*price/WAD), refused when the 512-bit product's high
// limb reaches WAD (the quotient would not fit) or cv exceeds MAX_INPUT.
func levMarkedValue(collateral, price *uint256.Int) (*uint256.Int, bool) {
	if price.IsZero() {
		return new(uint256.Int), false
	}
	hi, _ := levMul512(collateral, price)
	if !hi.Lt(uWad) {
		return new(uint256.Int), false
	}
	cv := levMulDivFloor(collateral, price, uWad)
	return cv, !cv.Gt(levMaxInput)
}

// levStrictAnchor is _strictAnchor -> (ok, anchor, h, halfLaw). The wall is inclusive; h is only set on the
// Hermite piece, where it is the least h in [H_JOIN, H_WALL] with debt*h <= cv*D(h).
func levStrictAnchor(cv, debt, rWad *uint256.Int) (bool, *uint256.Int, *uint256.Int, bool) {
	if !rWad.Eq(levLeverageRatioWad) || cv.Gt(levMaxInput) || debt.Gt(levMaxInput) {
		return false, new(uint256.Int), new(uint256.Int), false
	}
	if cv.IsZero() {
		return debt.IsZero(), new(uint256.Int), new(uint256.Int), true
	}
	if levProductGt(debt, levHWall, cv, levDWall) {
		return false, new(uint256.Int), new(uint256.Int), false
	}
	if !levProductGt(debt, levHJoin, cv, levDJoin) {
		anchor := levHalfLawAnchor(cv, debt)
		return !anchor.IsZero(), anchor, new(uint256.Int), true
	}
	lo, hi := new(uint256.Int).Set(levHJoin), new(uint256.Int).Set(levHWall)
	var mid uint256.Int
	for lo.Lt(hi) {
		mid.Add(lo, hi)
		mid.Rsh(&mid, 1)
		if levProductGt(debt, &mid, cv, levDebtNormWad(&mid)) {
			lo.AddUint64(&mid, 1)
		} else {
			hi.Set(&mid)
		}
	}
	anchor := levMulDivFloor(levMul(levThree, cv), uWad, levMul(levTwo, lo))
	return !anchor.IsZero(), anchor, lo, false
}

// levHalfLawAnchor is _halfLawAnchor: the upper root of 3(B+d)^2 = 8B*cv, floored by
// (8cv - 6d + 4*isqrt(2cv(2cv-3d)))/6 and then lifted by one wei when that still satisfies the root interval.
func levHalfLawAnchor(cv, debt *uint256.Int) *uint256.Int {
	twoCv := levMul(levTwo, cv)
	threeDebt := levMul(levThree, debt)
	if threeDebt.Gt(twoCv) {
		return new(uint256.Int)
	}
	root := new(uint256.Int).Sqrt(levMul(twoCv, levSub(twoCv, threeDebt)))
	anchor := levSub(levMul(levEight, cv), levMul(levSix, debt))
	anchor.Add(anchor, levMul(uint256.NewInt(4), root))
	anchor.Div(anchor, levSix)
	plusOne := levAdd(anchor, uOne)
	if levRootIntervalContains(cv, debt, plusOne) {
		return plusOne
	}
	return anchor
}

// levRootIntervalContains is _rootIntervalContains: !(3*(anchor+debt)^2 > 8*anchor*cv), with sum^2 and 8B
// formed at 256 bits and the outer products compared at 512.
func levRootIntervalContains(cv, debt, anchor *uint256.Int) bool {
	sum := levAdd(anchor, debt)
	return !levProductGt(levThree, levMul(sum, sum), levMul(levEight, anchor), cv)
}

// levDeleverageQuote is deleverageQuote -> (collateralOut, newCollateral, newDebt): retire stableIn of debt
// and release collateral with output-shrink rounding. A zero collateralOut is a refusal and returns the
// inputs unchanged. The strict branch prices the fee pro rata (_deleverageProRata); a solvent state beyond
// the wall takes the recovery continuation, which keeps the pre-fill cliff spread.
func levDeleverageQuote(collateral, debt, price, rWad, spreadPpm, stableIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	if collateral.IsZero() || price.IsZero() || !rWad.Eq(levLeverageRatioWad) || !spreadPpm.Lt(uPpm) ||
		stableIn.IsZero() || stableIn.Gt(debt) || debt.Gt(levMaxInput) {
		return levRefused(collateral, debt)
	}
	cv, marked := levMarkedValue(collateral, price)
	if !marked || cv.IsZero() {
		return levRefused(collateral, debt)
	}
	normal, anchor, _, _ := levStrictAnchor(cv, debt, rWad)
	if !normal || anchor.IsZero() {
		return levRecoveryDeleverage(collateral, debt, cv, price, rWad, spreadPpm, stableIn)
	}

	newDebt := levSub(debt, stableIn)
	feasible, cvRequired := levCvRequiredOnAnchor(anchor, newDebt)
	if !feasible {
		return levRefused(collateral, debt)
	}
	collateralRequired := levMulDivUp(cvRequired, uWad, price)
	if !collateralRequired.Lt(collateral) {
		return levRefused(collateral, debt)
	}
	outGross := levSub(collateral, collateralRequired)
	collateralOut, effectiveSpread := levDeleverageProRata(anchor, collateral, debt, newDebt, price, outGross, spreadPpm)
	if collateralOut.IsZero() {
		return levRefused(collateral, debt)
	}
	newCollateral := levSub(collateral, collateralOut)
	if !levPostStrictAnchorAccepted(anchor, newCollateral, newDebt, price, rWad, effectiveSpread) {
		return levRefused(collateral, debt)
	}
	return collateralOut, newCollateral, newDebt
}

// levLeverageQuote is leverageQuote -> (stableOut, newCollateral, newDebt): add collateralIn and borrow along
// the same fixed-anchor book, net of the spread (floor), with the NET payout booked as debt. Strict-only.
func levLeverageQuote(collateral, debt, price, rWad, spreadPpm, collateralIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	if collateral.IsZero() || price.IsZero() || !rWad.Eq(levLeverageRatioWad) || !spreadPpm.Lt(uPpm) ||
		collateralIn.IsZero() || collateralIn.Gt(levMaxInput) || debt.Gt(levMaxInput) ||
		collateralIn.Gt(levSub(maxUint256, collateral)) {
		return levRefused(collateral, debt)
	}
	cv, marked := levMarkedValue(collateral, price)
	if !marked || cv.IsZero() {
		return levRefused(collateral, debt)
	}
	normal, anchor, _, _ := levStrictAnchor(cv, debt, rWad)
	if !normal || anchor.IsZero() {
		return levRefused(collateral, debt)
	}

	newCollateral := levAdd(collateral, collateralIn)
	newCv, newMarked := levMarkedValue(newCollateral, price)
	if !newMarked || !newCv.Gt(cv) {
		return levRefused(collateral, debt)
	}
	feasible, debtCap := levDebtCapOnAnchor(anchor, newCv)
	if !feasible || !debtCap.Gt(debt) {
		return levRefused(collateral, debt)
	}
	grossOut := levSub(debtCap, debt)
	stableOut := levMulDivFloor(grossOut, levSub(uPpm, spreadPpm), uPpm)
	if stableOut.IsZero() {
		return levRefused(collateral, debt)
	}
	newDebt := levAdd(debt, stableOut)
	if !levPostStrictAnchorAccepted(anchor, newCollateral, newDebt, price, rWad, spreadPpm) {
		return levRefused(collateral, debt)
	}
	return stableOut, newCollateral, newDebt
}

// levCvRequiredOnAnchor is _cvRequiredOnAnchor: the least marked value keeping `anchor` after retiring to
// `debt`. Half-law piece (3d*WAD <= 2B*D_JOIN): ceil(3(B+d)^2/(8B)); Hermite piece: the least h with
// 3d*WAD <= 2B*D(h), then ceil(2B*h/(3*WAD)); past the wall it is infeasible.
func levCvRequiredOnAnchor(anchor, debt *uint256.Int) (bool, *uint256.Int) {
	threeDebt := levMul(levThree, debt)
	twoAnchor := levMul(levTwo, anchor)
	if !levProductGt(threeDebt, uWad, twoAnchor, levDJoin) {
		sum := levAdd(anchor, debt)
		return true, levMulDivUp(levThree, levMul(sum, sum), levMul(levEight, anchor))
	}
	if levProductGt(threeDebt, uWad, twoAnchor, levDWall) {
		return false, new(uint256.Int)
	}
	lo, hi := new(uint256.Int).Set(levHJoin), new(uint256.Int).Set(levHWall)
	var mid uint256.Int
	for lo.Lt(hi) {
		mid.Add(lo, hi)
		mid.Rsh(&mid, 1)
		if levProductGt(threeDebt, uWad, twoAnchor, levDebtNormWad(&mid)) {
			lo.AddUint64(&mid, 1)
		} else {
			hi.Set(&mid)
		}
	}
	return true, levMulDivUp(twoAnchor, lo, levThreeWad)
}

// levDebtCapOnAnchor is _debtCapOnAnchor: the greatest debt whose C1 anchor at cv is at least `anchor`, with
// h = floor(3cv*WAD/(2B)) confined to [H_ZERO, H_WALL]; isqrt(floor(8B*cv/3)) - B on the half law.
func levDebtCapOnAnchor(anchor, cv *uint256.Int) (bool, *uint256.Int) {
	h := levMulDivFloor(levMul(levThree, cv), uWad, levMul(levTwo, anchor))
	if h.Lt(levHZero) || h.Gt(levHWall) {
		return false, new(uint256.Int)
	}
	if !h.Gt(levHJoin) {
		root := new(uint256.Int).Sqrt(levMulDivFloor(levMul(levEight, anchor), cv, levThree))
		if !root.Gt(anchor) {
			return false, new(uint256.Int)
		}
		return true, root.Sub(root, anchor)
	}
	return true, levMulDivFloor(cv, levDebtNormWad(h), h)
}

// levRecoveryDeleverage is _recoveryDeleverage: a missed-wall state retires debt along its conservative
// continuation on the wall anchor. The partial retirement's y is the least y in [0, y0(+1)] with
// D(y) >= newDebt; the whole release is charged the PRE-fill cliff spread (_deleverageSpread).
func levRecoveryDeleverage(collateral, debt, cv, price, rWad, spreadPpm, stableIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	recovery := levRecoveryStateFor(cv, debt, rWad)
	if !recovery.ok || stableIn.Gt(recovery.stableToWall) {
		return levRefused(collateral, debt)
	}

	newDebt := levSub(debt, stableIn)
	yNew := new(uint256.Int)
	if !stableIn.Eq(recovery.stableToWall) {
		lo := new(uint256.Int)
		hi := new(uint256.Int).Set(recovery.y)
		if recovery.y.Lt(levWadMinusOne) {
			hi.AddUint64(recovery.y, 1)
		}
		if levRecoveryDebtAtY(recovery.wallCv, hi).Lt(newDebt) {
			return levRefused(collateral, debt)
		}
		var mid uint256.Int
		for lo.Lt(hi) {
			mid.Add(lo, hi)
			mid.Rsh(&mid, 1)
			if levRecoveryDebtAtY(recovery.wallCv, &mid).Lt(newDebt) {
				lo.AddUint64(&mid, 1)
			} else {
				hi.Set(&mid)
			}
		}
		yNew = lo
	}

	invariantCv := recovery.wallCv
	if !yNew.IsZero() {
		invariantCv = levMulDivUp(recovery.wallCv, levAdd(uWad, yNew), levSub(uWad, yNew))
	}
	collateralRequired := levMulDivUp(invariantCv, uWad, price)
	if !collateralRequired.Lt(collateral) {
		return levRefused(collateral, debt)
	}
	outGross := levSub(collateral, collateralRequired)
	effectiveSpread := levDeleverageSpread(cv, debt, spreadPpm)
	collateralOut := levMulDivFloor(outGross, levSub(uPpm, effectiveSpread), uPpm)
	if collateralOut.IsZero() {
		return levRefused(collateral, debt)
	}
	newCollateral := levSub(collateral, collateralOut)
	if !levPostAnyAnchorAccepted(recovery.anchor, newCollateral, newDebt, price, rWad, effectiveSpread) {
		return levRefused(collateral, debt)
	}
	return collateralOut, newCollateral, newDebt
}

// levRecoveryDebtAtY is _recoveryDebtAtY: z = ceil(wallCv(1+y)/(1-y)) (wallCv at y = 0) times
// rho(y) = (D_WALL + (H_WALL-D_WALL)y^2)/H_WALL, floored.
func levRecoveryDebtAtY(wallCv, y *uint256.Int) *uint256.Int {
	z := wallCv
	if !y.IsZero() {
		z = levMulDivUp(wallCv, levAdd(uWad, y), levSub(uWad, y))
	}
	rhoNumerator := levMul(levHWallMinusDWall, y)
	rhoNumerator.Mul(rhoNumerator, y)
	rhoNumerator.Add(levRhoWallNumerator, rhoNumerator)
	return levMulDivFloor(z, rhoNumerator, levRhoDenominator)
}

// levDeleverageSpread is _deleverageSpread, the whole-fill cliff read off the PRE-fill state: the cap when
// 2*debt >= cv and the posted spread exceeds it. Reachable only from the recovery branch.
func levDeleverageSpread(cv, debt, postedSpread *uint256.Int) *uint256.Int {
	if !levMul(levTwo, debt).Lt(cv) && postedSpread.Gt(levTargetSpreadCapPpm) {
		return new(uint256.Int).Set(levTargetSpreadCapPpm)
	}
	return new(uint256.Int).Set(postedSpread)
}

// levDeleverageProRata is _deleverageProRata -> (out, effSpread): the cap is charged only on collateral
// released while the fill is at or below target, i.e. while 3*debt >= anchor, and the posted spread on the
// rest. The split is at T = (anchor+2)/3 with collAtTarget = ceil(cvRequired(anchor, T)*WAD/price); each leg
// floors separately and effSpread = posted - floor((posted-cap)*grossCapped/outGross) feeds the strict
// post-anchor gate.
func levDeleverageProRata(anchor, collateral, debt, newDebt, price, outGross, postedSpread *uint256.Int) (*uint256.Int, *uint256.Int) {
	posted := func() (*uint256.Int, *uint256.Int) {
		return levMulDivFloor(outGross, levSub(uPpm, postedSpread), uPpm), new(uint256.Int).Set(postedSpread)
	}
	// Nothing to blend: no cap configured, or the fill starts above target.
	if !postedSpread.Gt(levTargetSpreadCapPpm) || levMul(levThree, debt).Lt(anchor) {
		return posted()
	}
	// Dust guard: sub-101-wei anchors pay the posted rate.
	if anchor.Lt(levDustAnchorFloor) {
		return posted()
	}
	// The whole fill stays at or below target.
	if !levMul(levThree, newDebt).Lt(anchor) {
		return levMulDivFloor(outGross, levSub(uPpm, levTargetSpreadCapPpm), uPpm),
			new(uint256.Int).Set(levTargetSpreadCapPpm)
	}
	// The fill crosses out: split the release at the crossing debt T.
	ok, cvAtTarget := levCvRequiredOnAnchor(anchor, new(uint256.Int).Div(levAdd(anchor, levTwo), levThree))
	if !ok {
		return posted()
	}
	collAtTarget := levMulDivUp(cvAtTarget, uWad, price)
	if !collAtTarget.Lt(collateral) {
		return posted()
	}
	grossCapped := levSub(collateral, collAtTarget)
	if grossCapped.Gt(outGross) {
		grossCapped = outGross
	}
	out := levMulDivFloor(grossCapped, levSub(uPpm, levTargetSpreadCapPpm), uPpm)
	out.Add(out, levMulDivFloor(levSub(outGross, grossCapped), levSub(uPpm, postedSpread), uPpm))
	discount := levMul(levSub(postedSpread, levTargetSpreadCapPpm), grossCapped)
	discount.Div(discount, outGross)
	return out, levSub(postedSpread, discount)
}

// levPostStrictAnchorAccepted is _postStrictAnchorAccepted: the post-fill state must sit on the strict
// branch with an anchor that does not fall (spread 0) or strictly rises (any spread).
func levPostStrictAnchorAccepted(preAnchor, collateral, debt, price, rWad, spreadPpm *uint256.Int) bool {
	cv, marked := levMarkedValue(collateral, price)
	if !marked {
		return false
	}
	ok, postAnchor, _, _ := levStrictAnchor(cv, debt, rWad)
	if !ok {
		return false
	}
	if spreadPpm.IsZero() {
		return !postAnchor.Lt(preAnchor)
	}
	return postAnchor.Gt(preAnchor)
}

// levPostAnyAnchorAccepted is _postAnyAnchorAccepted: as the strict gate, on the best-effort anchor.
func levPostAnyAnchorAccepted(preAnchor, collateral, debt, price, rWad, spreadPpm *uint256.Int) bool {
	cv, marked := levMarkedValue(collateral, price)
	if !marked {
		return false
	}
	postAnchor := levAnchorBestEffortCv(cv, debt, rWad)
	if spreadPpm.IsZero() {
		return !postAnchor.Lt(preAnchor)
	}
	return postAnchor.Gt(preAnchor)
}

// levRecoveryState is CollRebalancerMath.RecoveryState.
type levRecoveryState struct {
	ok           bool
	anchor       *uint256.Int
	baseX        *uint256.Int
	y            *uint256.Int
	wallCv       *uint256.Int
	wallDebt     *uint256.Int
	stableToWall *uint256.Int
}

// levRecoveryStateFor is _recoveryState: for debt*H_WALL > cv*D_WALL with debt < cv,
// y = isqrt(floor((debt*H_WALL - cv*D_WALL)*WAD^2 / (cv*(H_WALL-D_WALL)))), wallCv = floor(cv(1-y)/(1+y)),
// wallDebt = floor(wallCv*D_WALL/H_WALL); the anchor is the strict anchor at the wall and
// baseX = debt + floor((cv-debt)*y/WAD).
func levRecoveryStateFor(cv, debt, rWad *uint256.Int) levRecoveryState {
	var recovery levRecoveryState
	if !rWad.Eq(levLeverageRatioWad) || cv.IsZero() || cv.Gt(levMaxInput) || debt.Gt(levMaxInput) ||
		!debt.Lt(cv) || !levProductGt(debt, levHWall, cv, levDWall) {
		return recovery
	}
	numerator := levSub(levMul(debt, levHWall), levMul(cv, levDWall))
	denominator := levMul(cv, levHWallMinusDWall)
	y := new(uint256.Int).Sqrt(levMulDivFloor(numerator, uWadSquared, denominator))
	if y.IsZero() || !y.Lt(uWad) {
		return recovery
	}
	recovery.y = y
	recovery.wallCv = levMulDivFloor(cv, levSub(uWad, y), levAdd(uWad, y))
	if recovery.wallCv.IsZero() {
		return recovery
	}
	recovery.wallDebt = levMulDivFloor(recovery.wallCv, levDWall, levHWall)
	if !recovery.wallDebt.Lt(debt) {
		return recovery
	}
	wallOk, wallAnchor, _, _ := levStrictAnchor(recovery.wallCv, recovery.wallDebt, rWad)
	if !wallOk || wallAnchor.IsZero() {
		return recovery
	}
	recovery.anchor = wallAnchor
	recovery.baseX = levAdd(debt, levMulDivFloor(levSub(cv, debt), y, uWad))
	recovery.stableToWall = levSub(debt, recovery.wallDebt)
	recovery.ok = true
	return recovery
}

// levAnchorBestEffortCv is anchorBestEffort(cv, debt, rWad): the strict anchor, else the recovery anchor of a
// solvent missed-wall state, else zero.
func levAnchorBestEffortCv(cv, debt, rWad *uint256.Int) *uint256.Int {
	if ok, anchor, _, _ := levStrictAnchor(cv, debt, rWad); ok {
		return anchor
	}
	if recovery := levRecoveryStateFor(cv, debt, rWad); recovery.ok {
		return recovery.anchor
	}
	return new(uint256.Int)
}

// levIsStateSafe is isStateSafe: a normal or recovery state whose anchor meets requiredXAnchor.
func levIsStateSafe(collateral, debt, price, requiredXAnchor, rWad *uint256.Int) bool {
	if !rWad.Eq(levLeverageRatioWad) {
		return false
	}
	cv, marked := levMarkedValue(collateral, price)
	if !marked {
		return false
	}
	if cv.IsZero() {
		return collateral.IsZero() && debt.IsZero() && requiredXAnchor.IsZero()
	}
	if ok, anchor, _, _ := levStrictAnchor(cv, debt, rWad); ok {
		return !anchor.IsZero() && !anchor.Lt(requiredXAnchor)
	}
	recovery := levRecoveryStateFor(cv, debt, rWad)
	return recovery.ok && !recovery.anchor.Lt(requiredXAnchor)
}

// levAnchorAndBase is anchorAndBase -> (xAnchor, baseX): the value anchor and the stable-value marginal
// numerator, (anchor+debt)/2 on the half law and floor(cv*phi(h)/WAD) on the Hermite piece; a missed-wall
// state reports its recovery pair and anything else (0, 0).
func levAnchorAndBase(collateral, debt, price, rWad *uint256.Int) (*uint256.Int, *uint256.Int) {
	cv, marked := levMarkedValue(collateral, price)
	if !marked {
		return new(uint256.Int), new(uint256.Int)
	}
	ok, anchor, h, halfLaw := levStrictAnchor(cv, debt, rWad)
	if ok {
		if anchor.IsZero() {
			return new(uint256.Int), new(uint256.Int)
		}
		if halfLaw {
			baseX := levAdd(anchor, debt)
			return anchor, baseX.Div(baseX, levTwo)
		}
		return anchor, levMulDivFloor(cv, levPhiWad(h), uWad)
	}
	if recovery := levRecoveryStateFor(cv, debt, rWad); recovery.ok {
		return recovery.anchor, recovery.baseX
	}
	return new(uint256.Int), new(uint256.Int)
}

// levPhiWad is _phiWad: WAD^2/isqrt(h*WAD) on the half law, the cubic Bezier at x = floor((h-H_JOIN)*WAD/WIDTH)
// on the transition.
func levPhiWad(h *uint256.Int) *uint256.Int {
	if !h.Gt(levHJoin) {
		root := new(uint256.Int).Sqrt(levMul(h, uWad))
		return root.Div(uWadSquared, root)
	}
	return levBezier3(levMulDivFloor(levSub(h, levHJoin), uWad, levWidth))
}

// levDebtNormWad is _debtNormWad: 2*isqrt(h*WAD) - 1.5 WAD on the half law, D_JOIN + floor(WIDTH*B4(x)/WAD) on
// the transition. Callers only reach the half-law piece at h >= H_JOIN, where the subtraction cannot wrap.
func levDebtNormWad(h *uint256.Int) *uint256.Int {
	if !h.Gt(levHJoin) {
		root := new(uint256.Int).Sqrt(levMul(h, uWad))
		root.Mul(root, levTwo)
		return root.Sub(root, levHalfLawOffset)
	}
	x := levMulDivFloor(levSub(h, levHJoin), uWad, levWidth)
	return levAdd(levDJoin, levMulDivFloor(levWidth, levBezier4(x), uWad))
}

// levLerpFloor is _lerpFloor: floor(a(1-x) + b*x) at WAD scale, ceiling the step on a descending leg.
func levLerpFloor(a, b, x *uint256.Int) *uint256.Int {
	if !b.Lt(a) {
		return levAdd(a, levMulDivFloor(levSub(b, a), x, uWad))
	}
	return levSub(a, levMulDivUp(levSub(a, b), x, uWad))
}

// levBezier3 is _bezier3, de Casteljau over P0..P3.
func levBezier3(x *uint256.Int) *uint256.Int {
	a0 := levLerpFloor(levP0, levP1, x)
	a1 := levLerpFloor(levP1, levP2, x)
	a2 := levLerpFloor(levP2, levP3, x)
	b0 := levLerpFloor(a0, a1, x)
	b1 := levLerpFloor(a1, a2, x)
	return levLerpFloor(b0, b1, x)
}

// levBezier4 is _bezier4, de Casteljau over Q0 = 0, Q1..Q4.
func levBezier4(x *uint256.Int) *uint256.Int {
	a0 := levLerpFloor(uZero, levQ1, x)
	a1 := levLerpFloor(levQ1, levQ2, x)
	a2 := levLerpFloor(levQ2, levQ3, x)
	a3 := levLerpFloor(levQ3, levQ4, x)
	b0 := levLerpFloor(a0, a1, x)
	b1 := levLerpFloor(a1, a2, x)
	b2 := levLerpFloor(a2, a3, x)
	c0 := levLerpFloor(b0, b1, x)
	c1 := levLerpFloor(b1, b2, x)
	return levLerpFloor(c0, c1, x)
}
