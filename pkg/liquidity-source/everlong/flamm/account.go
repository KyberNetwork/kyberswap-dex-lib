package everlongflamm

import (
	"github.com/holiman/uint256"
)

// MorphoBlueAccount (src/core/mm/MorphoBlueAccount.sol, the kind-0 financing account; c104's is
// 0x6760E3b032eE2d670Cb684d9076b8f48cb066c48): the accrued views the Router prices every leg from, and the
// normal-lane mutators as transitions on the tracked Morpho market. Views use OpenZeppelin Math.mulDiv
// (512-bit intermediates, reverting only when the quotient overflows), unlike Blue's checked-product MathLib;
// the two agree everywhere Blue itself does not revert.

var (
	mmIrmStaleGrace = uint256.NewInt(3600)                                          // IRM_STALE_GRACE
	mmGraceRateWad  = uint256.NewInt(8_000_000_000_000_000_000 / (365 * 24 * 3600)) // GRACE_RATE_WAD = 8e18 / 365 days
)

// mmMulDivOZ is OpenZeppelin (compat v4) Math.mulDiv. A product that fits 256 bits takes the checked `prod0 /
// denominator`, Panic(0x12) on a zero denominator; a wider one must pass `require(denominator > prod1)`, which a
// zero denominator fails too, and which is exactly "the quotient fits 256 bits" otherwise.
func mmMulDivOZ(x, y, d *uint256.Int) (uint256.Int, error) {
	if d.IsZero() {
		var p uint256.Int
		if _, overflow := p.MulOverflow(x, y); overflow {
			return uint256.Int{}, errMulDivOverflow
		}
		return uint256.Int{}, errPanicDivZero
	}
	z, overflow := mulDiv(x, y, d)
	if overflow {
		return *z, errMulDivOverflow
	}
	return *z, nil
}

// mmMulDivOZUp is Math.mulDiv(x, y, d, Rounding.Up): the floor, then a checked `+= 1` when mulmod(x, y, d) > 0
// (Panic(0x11) if the floor is already type(uint256).max).
func mmMulDivOZUp(x, y, d *uint256.Int) (uint256.Int, error) {
	z, err := mmMulDivOZ(x, y, d)
	if err != nil {
		return z, err
	}
	var rem uint256.Int
	if !rem.MulMod(x, y, d).IsZero() {
		if z.Eq(maxUint256) {
			return z, errPanicArithmetic
		}
		z.AddUint64(&z, 1)
	}
	return z, nil
}

// mmAccountTaylor is the account's _wTaylorCompounded: Blue's three terms, the squares through Math.mulDiv.
func mmAccountTaylor(x, n *uint256.Int) (uint256.Int, error) {
	var first, sum uint256.Int
	if _, overflow := first.MulOverflow(x, n); overflow {
		return sum, errPanicArithmetic
	}
	second, err := mmMulDivOZ(&first, &first, mmTwoWad)
	if err != nil {
		return sum, err
	}
	third, err := mmMulDivOZ(&second, &first, mmThreeWad)
	if err != nil {
		return sum, err
	}
	if _, overflow := sum.AddOverflow(&first, &second); overflow {
		return sum, errPanicArithmetic
	}
	if _, overflow := sum.AddOverflow(&sum, &third); overflow {
		return sum, errPanicArithmetic
	}
	return sum, nil
}

// mmAccruedTotals are the four market totals as `_state` reports them.
type mmAccruedTotals struct {
	Readable bool
	Tsa, Tss uint256.Int
	Tba, Tbs uint256.Int
}

// accountState is MorphoBlueAccount._state(id) at `now`: Blue's totals as its next _accrueInterest will leave them.
// Nothing accrues when no time has passed or nothing is borrowed. While the IRM is unreadable the figures stay the
// last accrued ones: beyond IRM_STALE_GRACE the venue is unreadable; inside it the DEBT leg alone is marked up at
// GRACE_RATE_WAD and the supply legs are frozen. Otherwise the interest lands on both legs and the fee shares
// are minted at the post-interest supply less the fee (Blue's toSharesDown with Math.mulDiv).
func (v *mmVenueMarket) accountState(now uint64) (mmAccruedTotals, error) {
	m := &v.Market
	s := mmAccruedTotals{Readable: true, Tsa: m.TotalSupplyAssets, Tss: m.TotalSupplyShares,
		Tba: m.TotalBorrowAssets, Tbs: m.TotalBorrowShares}
	var nowU uint256.Int
	nowU.SetUint64(now)
	if !nowU.Gt(&m.LastUpdate) || s.Tba.IsZero() {
		return s, nil
	}
	var elapsed uint256.Int
	elapsed.Sub(&nowU, &m.LastUpdate)
	ok, rate, err := v.tryBorrowRate(uZero, uZero, now)
	if err != nil {
		return s, err
	}
	if !ok {
		if elapsed.Gt(mmIrmStaleGrace) {
			s.Readable = false
			return s, nil
		}
		growth, err := mmAccountTaylor(mmGraceRateWad, &elapsed)
		if err != nil {
			return s, err
		}
		markup, err := mmMulDivOZ(&s.Tba, &growth, uWad)
		if err != nil {
			return s, err
		}
		if _, overflow := s.Tba.AddOverflow(&s.Tba, &markup); overflow {
			return s, errPanicArithmetic
		}
		return s, nil
	}
	growth, err := mmAccountTaylor(&rate, &elapsed)
	if err != nil {
		return s, err
	}
	interest, err := mmMulDivOZ(&s.Tba, &growth, uWad)
	if err != nil {
		return s, err
	}
	if _, overflow := s.Tba.AddOverflow(&s.Tba, &interest); overflow {
		return s, errPanicArithmetic
	}
	if _, overflow := s.Tsa.AddOverflow(&s.Tsa, &interest); overflow {
		return s, errPanicArithmetic
	}
	if !m.Fee.IsZero() {
		feeAmount, err := mmMulDivOZ(&interest, &m.Fee, uWad)
		if err != nil {
			return s, err
		}
		var vs, va uint256.Int
		vs.Add(&s.Tss, mmVirtualShares)
		va.Sub(&s.Tsa, &feeAmount).Add(&va, mmVirtualAssets)
		feeShares, err := mmMulDivOZ(&feeAmount, &vs, &va)
		if err != nil {
			return s, err
		}
		if _, overflow := s.Tss.AddOverflow(&s.Tss, &feeShares); overflow {
			return s, errPanicArithmetic
		}
	}
	return s, nil
}

// tryBorrowRate is MorphoBlueAccount._tryBorrowRate(id, deltaBorrow, deltaSupplyDown): the market's rate after
// `deltaBorrow` more debt and `deltaSupplyDown` less supply, read from the IRM view over the STORED (unaccrued)
// market with the borrow total saturated at uint128 max. A market with no IRM runs at exactly zero. A non-zero
// supply delta reaching the whole supply fails closed (no defined post-state); the supply shares are
// deliberately not scaled. An IRM revert (unreadable, or its own timestamp underflow) is `ok == false`. The
// only revert is the unchecked-looking `tba + deltaBorrow`, which is a checked uint256 add.
func (v *mmVenueMarket) tryBorrowRate(deltaBorrow, deltaSupplyDown *uint256.Int, now uint64) (bool, uint256.Int, error) {
	var rate uint256.Int
	if !v.HasIrm {
		return true, rate, nil
	}
	m := &v.Market
	if !deltaSupplyDown.IsZero() && !deltaSupplyDown.Lt(&m.TotalSupplyAssets) {
		return false, rate, nil
	}
	var nb uint256.Int
	if _, overflow := nb.AddOverflow(&m.TotalBorrowAssets, deltaBorrow); overflow {
		return false, rate, errPanicArithmetic
	}
	if nb.Gt(mmMaxUint128) {
		nb = *mmMaxUint128
	}
	if !v.IrmReadable {
		return false, rate, nil
	}
	s := *m
	s.TotalSupplyAssets.Sub(&m.TotalSupplyAssets, deltaSupplyDown)
	s.TotalBorrowAssets = nb
	rate, _, err := mmIrmBorrowRate(&s, &v.RateAtTarget, now)
	if err != nil {
		return false, uint256.Int{}, nil
	}
	return true, rate, nil
}

// mmVenueRead is IFinancingAccount.tryPosition(id).
type mmVenueRead struct {
	Readable     bool
	Collateral   uint256.Int
	SupplyShares uint256.Int
	Supplied     uint256.Int
	Debt         uint256.Int
}

// tryPosition is MorphoBlueAccount.tryPosition: the raw position valued at the `_state` totals, supply floored
// and debt ceiled; the figures are filled in even when the venue is unreadable.
func (v *mmVenueMarket) tryPosition(now uint64) (mmVenueRead, error) {
	r := mmVenueRead{Collateral: v.Position.Collateral, SupplyShares: v.Position.SupplyShares}
	s, err := v.accountState(now)
	if err != nil {
		return r, err
	}
	r.Readable = s.Readable
	if !r.SupplyShares.IsZero() {
		if r.Supplied, err = mmSharesToAssetsOZ(&r.SupplyShares, &s.Tsa, &s.Tss, false); err != nil {
			return r, err
		}
	}
	if !v.Position.BorrowShares.IsZero() {
		if r.Debt, err = mmSharesToAssetsOZ(&v.Position.BorrowShares, &s.Tba, &s.Tbs, true); err != nil {
			return r, err
		}
	}
	return r, nil
}

// mmSharesToAssetsOZ is `Math.mulDiv(shares, totalAssets + VIRTUAL_ASSETS, totalShares + VIRTUAL_SHARES[, Up])`.
func mmSharesToAssetsOZ(shares, totalAssets, totalShares *uint256.Int, up bool) (uint256.Int, error) {
	var va, vs uint256.Int
	if _, overflow := va.AddOverflow(totalAssets, mmVirtualAssets); overflow {
		return va, errPanicArithmetic
	}
	if _, overflow := vs.AddOverflow(totalShares, mmVirtualShares); overflow {
		return vs, errPanicArithmetic
	}
	if up {
		return mmMulDivOZUp(shares, &va, &vs)
	}
	return mmMulDivOZ(shares, &va, &vs)
}

// expectedState is MorphoBlueAccount._expectedState: `_state`, reverting IrmUnreadable past the grace.
func (v *mmVenueMarket) expectedState(now uint64) (mmAccruedTotals, error) {
	s, err := v.accountState(now)
	if err != nil {
		return s, err
	}
	if !s.Readable {
		return s, ErrIrmUnreadable
	}
	return s, nil
}

// debtOf is MorphoBlueAccount.debtOf / _debt: the borrow shares at the expected totals, rounded up.
func (v *mmVenueMarket) debtOf(now uint64) (uint256.Int, error) {
	if v.Position.BorrowShares.IsZero() {
		return uint256.Int{}, nil
	}
	s, err := v.expectedState(now)
	if err != nil {
		return uint256.Int{}, err
	}
	return mmSharesToAssetsOZ(&v.Position.BorrowShares, &s.Tba, &s.Tbs, true)
}

// supplySharesToAssets is MorphoBlueAccount.supplySharesToAssets: floor at the expected totals.
func (v *mmVenueMarket) supplySharesToAssets(shares *uint256.Int, now uint64) (uint256.Int, error) {
	if shares.IsZero() {
		return uint256.Int{}, nil
	}
	s, err := v.expectedState(now)
	if err != nil {
		return uint256.Int{}, err
	}
	return mmSharesToAssetsOZ(shares, &s.Tsa, &s.Tss, false)
}

// suppliedOf is MorphoBlueAccount.suppliedOf.
func (v *mmVenueMarket) suppliedOf(now uint64) (uint256.Int, error) {
	return v.supplySharesToAssets(&v.Position.SupplyShares, now)
}

// freeLiquidity is MorphoBlueAccount.freeLiquidity: the STORED totals' cash, never accrued.
func (v *mmVenueMarket) freeLiquidity() uint256.Int {
	var z uint256.Int
	if v.Market.TotalSupplyAssets.Gt(&v.Market.TotalBorrowAssets) {
		z.Sub(&v.Market.TotalSupplyAssets, &v.Market.TotalBorrowAssets)
	}
	return z
}

// borrowRateAfter is MorphoBlueAccount.borrowRateAfter.
func (v *mmVenueMarket) borrowRateAfter(deltaBorrow, deltaSupplyDown *uint256.Int, now uint64) (bool, uint256.Int, error) {
	return v.tryBorrowRate(deltaBorrow, deltaSupplyDown, now)
}

// ------------------------------------------------------------------ normal lane (Router-driven)

// accountSupplyCollateral is MorphoBlueAccount.supplyCollateral: Blue's supplyCollateral (no accrual).
func (v *mmVenueMarket) accountSupplyCollateral(assets *uint256.Int) error {
	return v.morphoSupplyCollateral(assets)
}

// accountWithdrawCollateral is MorphoBlueAccount.withdrawCollateral: Blue's withdrawCollateral (accrual, health).
func (v *mmVenueMarket) accountWithdrawCollateral(assets *uint256.Int, now uint64) error {
	return v.morphoWithdrawCollateral(assets, now)
}

// accountBorrow is MorphoBlueAccount.borrow: Blue's borrow(assets, 0) to the pool.
func (v *mmVenueMarket) accountBorrow(assets *uint256.Int, now uint64) error {
	_, err := v.morphoBorrow(assets, now)
	return err
}

// accountRepay is MorphoBlueAccount._repay with the account holding at least `assets` (the Router pulls the
// leg in first): accrue (a revert there is IrmUnreadable); nothing to do without shares or assets; the whole
// position is closed by SHARES when the leg covers the accrued debt (so no dust share survives), otherwise
// repaid by assets, which must burn at least one share (ExactDelta otherwise).
func (v *mmVenueMarket) accountRepay(assets *uint256.Int, now uint64) (uint256.Int, error) {
	var repaid uint256.Int
	if err := v.morphoAccrue(now); err != nil {
		return repaid, ErrIrmUnreadable
	}
	shares := v.Position.BorrowShares
	if shares.IsZero() || assets.IsZero() {
		return repaid, nil
	}
	owed, err := v.debtOf(now)
	if err != nil {
		return repaid, err
	}
	pay := *minU(assets, &owed)
	if pay.IsZero() {
		return repaid, nil
	}
	full := !pay.Lt(&owed)
	if full {
		repaid, _, err = v.morphoRepay(uZero, &shares, now)
	} else {
		repaid, _, err = v.morphoRepay(&pay, uZero, now)
	}
	if err != nil {
		return repaid, err
	}
	after := v.Position.BorrowShares
	if full && !after.IsZero() || !full && !after.Lt(&shares) {
		return repaid, ErrExactDelta
	}
	if !repaid.Eq(&pay) {
		return repaid, ErrExactDelta
	}
	return repaid, nil
}

// accountSupply is MorphoBlueAccount.supply: Blue's supply(assets, 0); returns the minted shares.
func (v *mmVenueMarket) accountSupply(assets *uint256.Int, now uint64) (uint256.Int, error) {
	return v.morphoSupply(assets, now)
}

// accountWithdraw is MorphoBlueAccount._withdraw: exactly one of assets / shares (InvalidConfig otherwise),
// accrue (IrmUnreadable on a revert), nothing out of an empty position, then Blue's withdraw.
func (v *mmVenueMarket) accountWithdraw(assets, shares *uint256.Int, now uint64) (uint256.Int, uint256.Int, error) {
	if assets.IsZero() == shares.IsZero() {
		return uint256.Int{}, uint256.Int{}, ErrInvalidConfig
	}
	if err := v.morphoAccrue(now); err != nil {
		return uint256.Int{}, uint256.Int{}, ErrIrmUnreadable
	}
	if v.Position.SupplyShares.IsZero() {
		return uint256.Int{}, uint256.Int{}, nil
	}
	return v.morphoWithdraw(assets, shares, now)
}
