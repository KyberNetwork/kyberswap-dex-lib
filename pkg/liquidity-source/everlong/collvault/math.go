package everlongcollvault

import (
	"math/big"
)

// Pure-integer port of the deployed CollateralRebalancer curve ("CR math") plus the
// CollateralRebalancerSwapper's token-leg preview:
//   - CollRebalancerMath.leverageQuote / deleverageQuote (strict/normal branch),
//   - Swapper._previewTokenAmounts (CollVault ERC-4626 -> ALM proportional split),
//   - the composed quoteLeverageAt / quoteDeleverageAt the swap legs settle by.
//
// Source of truth is the DEPLOYED bytecode (which diverges from the protocol repo);
// every function replicates Solidity semantics exactly: floor division, Math.mulDiv
// with Rounding.Up = ceil, Math.sqrt = floor, and Mul512.productGt as full-precision
// a*b > c*d. big.Int is used because intermediates exceed 256 bits (the contract uses a
// 512-bit library for them). Validated wei-exact against the on-chain oracle at a
// pinned block (see pool_simulator_test.go).
//
// The curve constants (wall anchors, Bézier controls, rescue spread) are STRATEGY
// PARAMS frozen into each deployment — they are per-chain configuration, never
// universal math.

var (
	bigWad      = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	bigPpm      = big.NewInt(1_000_000)
	bigMaxInput = new(big.Int).Exp(big.NewInt(10), big.NewInt(38), nil)
	bigBp       = big.NewInt(10_000)
)

// CurveParams are the deployed CollateralRebalancer's frozen curve constants.
type CurveParams struct {
	LeverageRatioWad   *big.Int    `json:"rWad"`
	HZero              *big.Int    `json:"hZero"`
	HJoin              *big.Int    `json:"hJoin"`
	HWall              *big.Int    `json:"hWall"`
	Width              *big.Int    `json:"width"`
	DJoin              *big.Int    `json:"dJoin"`
	DWall              *big.Int    `json:"dWall"`
	RescueSpreadPpm    *big.Int    `json:"rescueSpread"`
	BezierPhi          [4]*big.Int `json:"bezierPhi"`      // P0..P3: phi = dD/dh cubic controls
	BezierIntegral     [5]*big.Int `json:"bezierIntegral"` // Q0..Q4: exact quartic integral controls
	PhysicalCrFloorWad *big.Int    `json:"crFloor"`        // PRE-fill floor capping the max leverage lot
}

// VaultState is the per-refresh rebalancer + vault snapshot (all read from chain).
type VaultState struct {
	// exchangeState(): position collateral (CollVault shares), debt (stable wei),
	// reservation price R (WAD) and the posted spread (PPM).
	Collateral *big.Int `json:"c"`
	Debt       *big.Int `json:"d"`
	PriceWad   *big.Int `json:"r"`
	SpreadPpm  *big.Int `json:"s"`

	// ALM & CollVault state for the token-leg split.
	AlmStableReserve   *big.Int `json:"asr"` // alm.getTotalAmounts()[0]
	AlmVolatileReserve *big.Int `json:"avr"` // alm.getTotalAmounts()[1]
	AlmSupply          *big.Int `json:"as"`  // alm.totalSupply()
	CvTotalAssets      *big.Int `json:"cta"` // collVault.totalAssets()
	CvTotalSupply      *big.Int `json:"cts"` // collVault.totalSupply()
	CvDecimalsOffset   uint8    `json:"cdo"` // 18 - collVault.assetDecimals()
	WithdrawFeeBp      *big.Int `json:"wfb"` // collVault.getWithdrawFee()

	// alm.getReservesAtReference(): reference-marked physical reserves for the
	// PHYSICAL_CR_FLOOR check bounding the max leverage lot.
	RefStableReserve   *big.Int `json:"rsr"`
	RefAssetReserve    *big.Int `json:"rar"`
	RefRawReferenceWad *big.Int `json:"rrw"`
}

// ---------- Solidity primitive equivalents ----------

// mulDiv = Math.mulDiv(x, y, d): floor(x*y/d). Operands are non-negative.
func mulDiv(x, y, d *big.Int) *big.Int {
	var p big.Int
	p.Mul(x, y)
	return p.Quo(&p, d)
}

// mulDivUp = Math.mulDiv(x, y, d, Rounding.Up): ceil(x*y/d). Operands are non-negative.
func mulDivUp(x, y, d *big.Int) *big.Int {
	var p, m big.Int
	p.Mul(x, y)
	p.QuoRem(&p, d, &m)
	if m.Sign() != 0 {
		p.Add(&p, big.NewInt(1))
	}
	return &p
}

// productGt = Mul512.productGt: a*b > c*d at full precision.
func productGt(a, b, c, d *big.Int) bool {
	var l, r big.Int
	return l.Mul(a, b).Cmp(r.Mul(c, d)) > 0
}

// ---------- Marked value & anchor ----------

// markedValue: cv = floor(collateral*price/WAD); ok=false on overflow/out-of-bounds.
func markedValue(collateral, price *big.Int) (*big.Int, bool) {
	if price.Sign() == 0 {
		return nil, false
	}
	// the contract rejects when the 512-bit product's high word reaches WAD<<256
	var limit big.Int
	limit.Lsh(bigWad, 256)
	var p big.Int
	if p.Mul(collateral, price); p.Cmp(&limit) >= 0 {
		return nil, false
	}
	cv := mulDiv(collateral, price, bigWad)
	return cv, cv.Cmp(bigMaxInput) <= 0
}

func rootIntervalContains(cv, debt, anchor *big.Int) bool {
	var s, s2, a8 big.Int
	s.Add(anchor, debt)
	s2.Mul(&s, &s)
	a8.Mul(big.NewInt(8), anchor)
	return !productGt(big.NewInt(3), &s2, &a8, cv)
}

func halfLawAnchor(cv, debt *big.Int) *big.Int {
	var twoCv, threeDebt big.Int
	twoCv.Mul(big.NewInt(2), cv)
	threeDebt.Mul(big.NewInt(3), debt)
	if threeDebt.Cmp(&twoCv) > 0 {
		return new(big.Int)
	}
	var rootArg, root big.Int
	rootArg.Sub(&twoCv, &threeDebt)
	rootArg.Mul(&twoCv, &rootArg)
	root.Sqrt(&rootArg) // Math.sqrt default rounding = floor
	var anchor big.Int
	anchor.Mul(big.NewInt(8), cv)
	anchor.Sub(&anchor, new(big.Int).Mul(big.NewInt(6), debt))
	anchor.Add(&anchor, new(big.Int).Mul(big.NewInt(4), &root))
	anchor.Quo(&anchor, big.NewInt(6))
	var plusOne big.Int
	plusOne.Add(&anchor, big.NewInt(1))
	if rootIntervalContains(cv, debt, &plusOne) {
		return &plusOne
	}
	return &anchor
}

func (cp *CurveParams) debtNormWad(h *big.Int) *big.Int {
	if h.Cmp(cp.HJoin) <= 0 {
		var arg, root big.Int
		arg.Mul(h, bigWad)
		root.Sqrt(&arg)
		root.Mul(big.NewInt(2), &root)
		return root.Sub(&root, big.NewInt(1_500_000_000_000_000_000))
	}
	var x big.Int
	x.Sub(h, cp.HJoin)
	xNorm := mulDiv(&x, bigWad, cp.Width)
	return new(big.Int).Add(cp.DJoin, mulDiv(cp.Width, cp.bezier4(xNorm), bigWad))
}

// strictAnchor -> (anchor, h, halfLaw, ok).
func (cp *CurveParams) strictAnchor(cv, debt, rWad *big.Int) (*big.Int, *big.Int, bool, bool) {
	if rWad.Cmp(cp.LeverageRatioWad) != 0 || cv.Cmp(bigMaxInput) > 0 || debt.Cmp(bigMaxInput) > 0 {
		return nil, nil, false, false
	}
	if cv.Sign() == 0 {
		return new(big.Int), new(big.Int), true, debt.Sign() == 0
	}
	if productGt(debt, cp.HWall, cv, cp.DWall) {
		return nil, nil, false, false
	}
	if !productGt(debt, cp.HJoin, cv, cp.DJoin) {
		anchor := halfLawAnchor(cv, debt)
		return anchor, new(big.Int), true, anchor.Sign() != 0
	}
	lo, hi := new(big.Int).Set(cp.HJoin), new(big.Int).Set(cp.HWall)
	var mid big.Int
	for lo.Cmp(hi) < 0 {
		mid.Add(lo, hi)
		mid.Rsh(&mid, 1)
		if productGt(debt, &mid, cv, cp.debtNormWad(&mid)) {
			lo.Add(&mid, big.NewInt(1))
		} else {
			hi.Set(&mid)
		}
	}
	var threeCv big.Int
	threeCv.Mul(big.NewInt(3), cv)
	var twoH big.Int
	twoH.Mul(big.NewInt(2), lo)
	anchor := mulDiv(&threeCv, bigWad, &twoH)
	return anchor, lo, false, anchor.Sign() != 0
}

// ---------- Normalized curve (Bézier) ----------

func lerpFloor(a, b, x *big.Int) *big.Int {
	if b.Cmp(a) >= 0 {
		var d big.Int
		d.Sub(b, a)
		return new(big.Int).Add(a, mulDiv(&d, x, bigWad))
	}
	var d big.Int
	d.Sub(a, b)
	return new(big.Int).Sub(a, mulDivUp(&d, x, bigWad))
}

func (cp *CurveParams) bezier4(x *big.Int) *big.Int {
	q := cp.BezierIntegral
	a0 := lerpFloor(q[0], q[1], x)
	a1 := lerpFloor(q[1], q[2], x)
	a2 := lerpFloor(q[2], q[3], x)
	a3 := lerpFloor(q[3], q[4], x)
	b0 := lerpFloor(a0, a1, x)
	b1 := lerpFloor(a1, a2, x)
	b2 := lerpFloor(a2, a3, x)
	c0 := lerpFloor(b0, b1, x)
	c1 := lerpFloor(b1, b2, x)
	return lerpFloor(c0, c1, x)
}

// ---------- Debt-cap / cv-required on an anchor ----------

func (cp *CurveParams) debtCapOnAnchor(anchor, cv *big.Int) (*big.Int, bool) {
	var threeCv, twoAnchor big.Int
	threeCv.Mul(big.NewInt(3), cv)
	twoAnchor.Mul(big.NewInt(2), anchor)
	h := mulDiv(&threeCv, bigWad, &twoAnchor)
	if h.Cmp(cp.HZero) < 0 || h.Cmp(cp.HWall) > 0 {
		return nil, false
	}
	if h.Cmp(cp.HJoin) <= 0 {
		var eightAnchor, root big.Int
		eightAnchor.Mul(big.NewInt(8), anchor)
		root.Sqrt(mulDiv(&eightAnchor, cv, big.NewInt(3)))
		if root.Cmp(anchor) <= 0 {
			return nil, false
		}
		return root.Sub(&root, anchor), true
	}
	return mulDiv(cv, cp.debtNormWad(h), h), true
}

func (cp *CurveParams) cvRequiredOnAnchor(anchor, debt *big.Int) (*big.Int, bool) {
	var threeDebt, twoAnchor big.Int
	threeDebt.Mul(big.NewInt(3), debt)
	twoAnchor.Mul(big.NewInt(2), anchor)
	if !productGt(&threeDebt, bigWad, &twoAnchor, cp.DJoin) {
		var s, s2 big.Int
		s.Add(anchor, debt)
		s2.Mul(&s, &s)
		var eightAnchor big.Int
		eightAnchor.Mul(big.NewInt(8), anchor)
		return mulDivUp(big.NewInt(3), &s2, &eightAnchor), true
	}
	if productGt(&threeDebt, bigWad, &twoAnchor, cp.DWall) {
		return nil, false
	}
	lo, hi := new(big.Int).Set(cp.HJoin), new(big.Int).Set(cp.HWall)
	var mid big.Int
	for lo.Cmp(hi) < 0 {
		mid.Add(lo, hi)
		mid.Rsh(&mid, 1)
		if productGt(&threeDebt, bigWad, &twoAnchor, cp.debtNormWad(&mid)) {
			lo.Add(&mid, big.NewInt(1))
		} else {
			hi.Set(&mid)
		}
	}
	var threeWad big.Int
	threeWad.Mul(big.NewInt(3), bigWad)
	return mulDivUp(&twoAnchor, lo, &threeWad), true
}

func (cp *CurveParams) deleverageSpread(cv, debt, posted *big.Int) *big.Int {
	var half big.Int
	half.Add(cv, big.NewInt(1))
	half.Quo(&half, big.NewInt(2))
	if debt.Cmp(&half) >= 0 && posted.Cmp(cp.RescueSpreadPpm) > 0 {
		return cp.RescueSpreadPpm
	}
	return posted
}

func (cp *CurveParams) postStrictAnchorAccepted(preAnchor, collateral, debt, price, rWad,
	spread *big.Int) bool {
	cv, ok := markedValue(collateral, price)
	if !ok {
		return false
	}
	postAnchor, _, _, aok := cp.strictAnchor(cv, debt, rWad)
	if !aok || (cv.Sign() != 0 && postAnchor.Sign() == 0) {
		return false
	}
	if spread.Sign() == 0 {
		return postAnchor.Cmp(preAnchor) >= 0
	}
	return postAnchor.Cmp(preAnchor) > 0
}

// ---------- Region tracking ----------

type region int

const (
	regionStrict region = iota
	// regionRecovery: past the honest CR wall — the recovery branch is intentionally NOT
	// ported (matching the reference port); the simulator refuses to quote there.
	regionRecovery
	regionOut
)

func (cp *CurveParams) stateRegion(collateral, debt, price *big.Int) region {
	cv, ok := markedValue(collateral, price)
	if !ok || cv.Sign() == 0 {
		return regionOut
	}
	if productGt(debt, cp.HWall, cv, cp.DWall) {
		return regionRecovery
	}
	anchor, _, _, aok := cp.strictAnchor(cv, debt, cp.LeverageRatioWad)
	if aok && anchor.Sign() != 0 {
		return regionStrict
	}
	return regionOut
}

// ---------- Public quotes (strict branch) ----------

// leverageQuote = CollRebalancerMath.leverageQuote -> (stableOut, newCollateral,
// newDebt). stableOut == 0 means the fill is rejected.
func (cp *CurveParams) leverageQuote(collateral, debt, price, rWad, spread,
	collateralIn *big.Int) (*big.Int, *big.Int, *big.Int) {
	reject := func() (*big.Int, *big.Int, *big.Int) { return new(big.Int), collateral, debt }
	if collateral.Sign() == 0 || price.Sign() == 0 || rWad.Cmp(cp.LeverageRatioWad) != 0 ||
		spread.Cmp(bigPpm) >= 0 || collateralIn.Sign() == 0 ||
		collateralIn.Cmp(bigMaxInput) > 0 || debt.Cmp(bigMaxInput) > 0 {
		return reject()
	}
	cv, ok := markedValue(collateral, price)
	if !ok || cv.Sign() == 0 {
		return reject()
	}
	anchor, _, _, aok := cp.strictAnchor(cv, debt, rWad)
	if !aok || anchor.Sign() == 0 {
		return reject()
	}
	newCollateral := new(big.Int).Add(collateral, collateralIn)
	newCv, ok := markedValue(newCollateral, price)
	if !ok || newCv.Cmp(cv) <= 0 {
		return reject()
	}
	debtCap, feasible := cp.debtCapOnAnchor(anchor, newCv)
	if !feasible || debtCap.Cmp(debt) <= 0 {
		return reject()
	}
	var grossOut, keep big.Int
	grossOut.Sub(debtCap, debt)
	keep.Sub(bigPpm, spread)
	stableOut := mulDiv(&grossOut, &keep, bigPpm)
	if stableOut.Sign() == 0 {
		return reject()
	}
	newDebt := new(big.Int).Add(debt, stableOut)
	if !cp.postStrictAnchorAccepted(anchor, newCollateral, newDebt, price, rWad, spread) {
		return reject()
	}
	return stableOut, newCollateral, newDebt
}

// deleverageQuote = CollRebalancerMath.deleverageQuote -> (collateralOut, newCollateral,
// newDebt, inRecovery). collateralOut == 0 means rejected; inRecovery means the state
// needs the unported recovery branch — the caller must refuse to quote.
func (cp *CurveParams) deleverageQuote(collateral, debt, price, rWad, spread,
	stableIn *big.Int) (*big.Int, *big.Int, *big.Int, bool) {
	reject := func() (*big.Int, *big.Int, *big.Int, bool) { return new(big.Int), collateral, debt, false }
	if collateral.Sign() == 0 || price.Sign() == 0 || rWad.Cmp(cp.LeverageRatioWad) != 0 ||
		spread.Cmp(bigPpm) >= 0 || stableIn.Sign() == 0 || stableIn.Cmp(debt) > 0 ||
		stableIn.Cmp(bigMaxInput) > 0 || debt.Cmp(bigMaxInput) > 0 {
		return reject()
	}
	cv, ok := markedValue(collateral, price)
	if !ok || cv.Sign() == 0 {
		return reject()
	}
	anchor, _, _, aok := cp.strictAnchor(cv, debt, rWad)
	if !aok || anchor.Sign() == 0 {
		return nil, nil, nil, true // recovery branch — not ported
	}
	newDebt := new(big.Int).Sub(debt, stableIn)
	cvRequired, feasible := cp.cvRequiredOnAnchor(anchor, newDebt)
	if !feasible {
		return reject()
	}
	collateralRequired := mulDivUp(cvRequired, bigWad, price)
	if collateralRequired.Cmp(collateral) >= 0 {
		return reject()
	}
	var outGross, keep big.Int
	outGross.Sub(collateral, collateralRequired)
	effSpread := cp.deleverageSpread(cv, debt, spread)
	keep.Sub(bigPpm, effSpread)
	collateralOut := mulDiv(&outGross, &keep, bigPpm)
	if collateralOut.Sign() == 0 {
		return reject()
	}
	newCollateral := new(big.Int).Sub(collateral, collateralOut)
	if !cp.postStrictAnchorAccepted(anchor, newCollateral, newDebt, price, rWad, effSpread) {
		return reject()
	}
	return collateralOut, newCollateral, newDebt, false
}

// ---------- previewTokenAmounts (CollVault 4626 -> ALM proportional split) ----------

// cvConvertToAssets = ERC4626Upgradeable._convertToAssets:
// shares*(totalAssets+1)/(totalSupply+10^offset) with the given rounding.
func (s *VaultState) cvConvertToAssets(shares *big.Int, up bool) *big.Int {
	var num, den big.Int
	num.Add(s.CvTotalAssets, big.NewInt(1))
	den.Exp(big.NewInt(10), big.NewInt(int64(s.CvDecimalsOffset)), nil)
	den.Add(s.CvTotalSupply, &den)
	if up {
		return mulDivUp(shares, &num, &den)
	}
	return mulDiv(shares, &num, &den)
}

// previewTokenAmounts = Swapper._previewTokenAmounts -> (stable, volatile, ok);
// ok=false is the NothingToFill revert.
func (s *VaultState) previewTokenAmounts(collVaultShares *big.Int, mint bool) (*big.Int, *big.Int, bool) {
	var almShares *big.Int
	if mint {
		almShares = s.cvConvertToAssets(collVaultShares, true) // previewMint
	} else {
		shareFee := mulDivUp(collVaultShares, s.WithdrawFeeBp, bigBp) // feeOnRaw, Up
		var net big.Int
		net.Sub(collVaultShares, shareFee)
		almShares = s.cvConvertToAssets(&net, false) // previewRedeem
	}
	if s.AlmSupply.Sign() == 0 || almShares.Sign() == 0 {
		return nil, nil, false
	}
	if mint {
		return mulDivUp(s.AlmStableReserve, almShares, s.AlmSupply),
			mulDivUp(s.AlmVolatileReserve, almShares, s.AlmSupply), true
	}
	return mulDiv(s.AlmStableReserve, almShares, s.AlmSupply),
		mulDiv(s.AlmVolatileReserve, almShares, s.AlmSupply), true
}

// ---------- Composite quotes (the values swap legs settle by) ----------

// quoteLeverageAt: net stable (KAI) the caller receives for minting sharesIn CollVault
// shares — gross debt draw minus the stable leg pulled to mint the shares.
func (cp *CurveParams) quoteLeverageAt(s *VaultState, sharesIn *big.Int) (*big.Int, bool) {
	grossStableOut, _, _ := cp.leverageQuote(s.Collateral, s.Debt, s.PriceWad,
		cp.LeverageRatioWad, s.SpreadPpm, sharesIn)
	stableIn, _, ok := s.previewTokenAmounts(sharesIn, true)
	if !ok {
		return nil, false
	}
	if grossStableOut.Cmp(stableIn) > 0 {
		return new(big.Int).Sub(grossStableOut, stableIn), true
	}
	return new(big.Int), true
}

// deleverageLegsAt: both freed legs at a gross stableDebtIn -> (stableOut, volatileOut).
// The net the caller fronts is stableDebtIn - stableOut. ok=false when unpriceable
// (recovery region or degenerate preview).
func (cp *CurveParams) deleverageLegsAt(s *VaultState, stableDebtIn *big.Int) (*big.Int, *big.Int, bool) {
	sharesOut, _, _, recovery := cp.deleverageQuote(s.Collateral, s.Debt, s.PriceWad,
		cp.LeverageRatioWad, s.SpreadPpm, stableDebtIn)
	if recovery || sharesOut.Sign() == 0 {
		return nil, nil, false
	}
	return s.previewTokenAmounts(sharesOut, false)
}

// ---------- Max-lot sizing ----------

// leverageOk: a leverage fill of collateralIn both quotes AND leaves the position above
// the physical-CR floor (the PRE-fill check bounding the max lot).
func (cp *CurveParams) leverageOk(s *VaultState, collateralIn, totalPhysicalValue *big.Int) bool {
	out, newColl, newDebt := cp.leverageQuote(s.Collateral, s.Debt, s.PriceWad,
		cp.LeverageRatioWad, s.SpreadPpm, collateralIn)
	if out.Sign() == 0 {
		return false
	}
	if newDebt.Sign() == 0 {
		return true
	}
	almShares := s.cvConvertToAssets(newColl, false)
	positionPhysicalValue := mulDiv(totalPhysicalValue, almShares, s.AlmSupply)
	requiredValue := mulDivUp(newDebt, cp.PhysicalCrFloorWad, bigWad)
	return positionPhysicalValue.Cmp(requiredValue) >= 0
}

// maxLeverageShares: largest collateralIn that fills and clears the physical-CR floor.
// Boundary bisection; zero when even size 1 fails.
func (cp *CurveParams) maxLeverageShares(s *VaultState) *big.Int {
	if s.AlmSupply.Sign() == 0 || s.RefRawReferenceWad.Sign() == 0 {
		return new(big.Int)
	}
	totalPhysicalValue := new(big.Int).Add(s.RefStableReserve,
		mulDiv(s.RefAssetReserve, s.RefRawReferenceWad, bigWad))
	one := big.NewInt(1)
	if !cp.leverageOk(s, one, totalPhysicalValue) {
		return new(big.Int)
	}
	lo := new(big.Int).Set(one)
	hi := mulDiv(bigMaxInput, bigWad, s.PriceWad)
	var mid, span big.Int
	for lo.Cmp(hi) < 0 {
		span.Sub(hi, lo)
		span.Add(&span, one)
		span.Quo(&span, big.NewInt(2))
		mid.Add(lo, &span)
		if cp.leverageOk(s, &mid, totalPhysicalValue) {
			lo.Set(&mid)
		} else {
			hi.Sub(&mid, one)
		}
	}
	return lo
}

// maxDeleverageIn: largest gross stableIn that still quotes. nil when the state needs
// the unported recovery branch.
func (cp *CurveParams) maxDeleverageIn(s *VaultState) *big.Int {
	if s.Debt.Sign() == 0 {
		return new(big.Int)
	}
	one := big.NewInt(1)
	lo, hi := new(big.Int).Set(one), new(big.Int).Set(s.Debt)
	var mid, span big.Int
	for lo.Cmp(hi) < 0 {
		span.Sub(hi, lo)
		span.Add(&span, one)
		span.Quo(&span, big.NewInt(2))
		mid.Add(lo, &span)
		out, _, _, recovery := cp.deleverageQuote(s.Collateral, s.Debt, s.PriceWad,
			cp.LeverageRatioWad, s.SpreadPpm, &mid)
		if recovery {
			return nil
		}
		if out.Sign() != 0 {
			lo.Set(&mid)
		} else {
			hi.Sub(&mid, one)
		}
	}
	// even size 1 may not quote
	out, _, _, recovery := cp.deleverageQuote(s.Collateral, s.Debt, s.PriceWad,
		cp.LeverageRatioWad, s.SpreadPpm, lo)
	if recovery {
		return nil
	}
	if out.Sign() == 0 {
		return new(big.Int)
	}
	return lo
}

// ---------- Inversions used by CalcAmountOut ----------

// sharesForVolatileIn: largest sharesIn whose mint-leg volatile requirement fits within
// amountIn (previewTokenAmounts' volatile leg is monotone nondecreasing in shares).
// The result is additionally capped by maxShares (the physical-CR-floor cap).
func (s *VaultState) sharesForVolatileIn(amountIn, maxShares *big.Int) *big.Int {
	if amountIn.Sign() <= 0 || maxShares.Sign() <= 0 {
		return new(big.Int)
	}
	volAtOne, ok := volatileForShares(s, big.NewInt(1))
	if !ok || volAtOne.Cmp(amountIn) > 0 {
		return new(big.Int)
	}
	one := big.NewInt(1)
	lo, hi := new(big.Int).Set(one), new(big.Int).Set(maxShares)
	var mid, span big.Int
	for lo.Cmp(hi) < 0 {
		span.Sub(hi, lo)
		span.Add(&span, one)
		span.Quo(&span, big.NewInt(2))
		mid.Add(lo, &span)
		vol, ok := volatileForShares(s, &mid)
		if ok && vol.Cmp(amountIn) <= 0 {
			lo.Set(&mid)
		} else {
			hi.Sub(&mid, one)
		}
	}
	return lo
}

func volatileForShares(s *VaultState, shares *big.Int) (*big.Int, bool) {
	_, volatile, ok := s.previewTokenAmounts(shares, true)
	if !ok {
		return nil, false
	}
	return volatile, true
}

// grossForNetStableIn: largest gross stableDebtIn (within [1, maxGross], where maxGross
// must already be a VALID quoting size — pass maxDeleverageIn) whose NET front
// (gross - freed stable) fits within netBudget. The net is nondecreasing in the gross,
// so the predicate is monotone; sub-wei grosses that round to a rejected quote are
// treated by their gross (net <= gross always), and the winner is re-validated.
func (cp *CurveParams) grossForNetStableIn(s *VaultState, netBudget, maxGross *big.Int) *big.Int {
	if netBudget.Sign() <= 0 || maxGross.Sign() <= 0 {
		return new(big.Int)
	}
	fitsNet := func(gross *big.Int) bool {
		stableOut, _, ok := cp.deleverageLegsAt(s, gross)
		if !ok {
			// only tiny grosses are invalid below maxGross; net <= gross bounds them
			return gross.Cmp(netBudget) <= 0
		}
		var net big.Int
		net.Sub(gross, stableOut)
		return net.Cmp(netBudget) <= 0
	}
	one := big.NewInt(1)
	lo, hi := new(big.Int).Set(one), new(big.Int).Set(maxGross)
	var mid, span big.Int
	for lo.Cmp(hi) < 0 {
		span.Sub(hi, lo)
		span.Add(&span, one)
		span.Quo(&span, big.NewInt(2))
		mid.Add(lo, &span)
		if fitsNet(&mid) {
			lo.Set(&mid)
		} else {
			hi.Sub(&mid, one)
		}
	}
	if _, _, ok := cp.deleverageLegsAt(s, lo); !ok {
		return new(big.Int)
	}
	return lo
}

// ---------- Direction / full-lot views (parity targets vs the on-chain lens) ----------

// rebalanceState: the pending realignment direction (0 none / 1 leverage / 2
// deleverage) and the MAX lot (in / out). ok=false when the state needs the unported
// recovery branch.
func (cp *CurveParams) rebalanceState(s *VaultState) (int, *big.Int, *big.Int, bool) {
	zero := func() (int, *big.Int, *big.Int, bool) { return 0, new(big.Int), new(big.Int), true }
	if s.Collateral.Sign() == 0 || s.PriceWad.Sign() == 0 {
		return zero()
	}
	one := big.NewInt(1)
	levProbe, _, _ := cp.leverageQuote(s.Collateral, s.Debt, s.PriceWad, cp.LeverageRatioWad, s.SpreadPpm, one)
	if levProbe.Sign() != 0 {
		x := cp.maxLeverageShares(s)
		if x.Sign() == 0 {
			return zero()
		}
		stableOut, _, _ := cp.leverageQuote(s.Collateral, s.Debt, s.PriceWad, cp.LeverageRatioWad, s.SpreadPpm, x)
		return 1, x, stableOut, true
	}
	if s.Debt.Sign() != 0 {
		probe, _, _, recovery := cp.deleverageQuote(s.Collateral, s.Debt, s.PriceWad,
			cp.LeverageRatioWad, s.SpreadPpm, one)
		if recovery {
			return 0, nil, nil, false
		}
		if probe.Sign() != 0 {
			sIn := cp.maxDeleverageIn(s)
			if sIn == nil {
				return 0, nil, nil, false
			}
			if sIn.Sign() == 0 {
				return zero()
			}
			out, _, _, recovery := cp.deleverageQuote(s.Collateral, s.Debt, s.PriceWad,
				cp.LeverageRatioWad, s.SpreadPpm, sIn)
			if recovery {
				return 0, nil, nil, false
			}
			return 2, sIn, out, true
		}
	}
	return zero()
}
