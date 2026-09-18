package everlongflamm

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Morpho Blue (0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb on Base) as the FLAMM financing account drives it:
// MathLib, SharesMathLib and the market transitions of src/Morpho.sol (morpho-org/morpho-blue v1.0.0, the
// verified singleton). Every function keeps Solidity's operation order: MathLib.mulDivDown is `(x*y)/d` with a
// CHECKED 256-bit product (a false `ok` is the arithmetic panic Morpho reverts with), mulDivUp is
// `(x*y + (d-1))/d`, and every uint128 total is re-checked on write (UtilsLib.toUint128 plus the checked
// uint128 add). Validated against the deployed singleton on Base forks (mm_*.json fixtures).

var (
	mmVirtualShares    = uint256.NewInt(1_000_000)               // SharesMathLib.VIRTUAL_SHARES
	mmVirtualAssets    = uOne                                    // SharesMathLib.VIRTUAL_ASSETS
	mmOraclePriceScale = new(uint256.Int).Set(big256.TenPow(36)) // ConstantsLib.ORACLE_PRICE_SCALE
	mmMaxUint128       = new(uint256.Int).Set(big256.UMaxU128)
	mmTwoWad           = new(uint256.Int).Mul(uWad, big256.U2)
	mmThreeWad         = new(uint256.Int).Mul(uWad, big256.U3)
)

// Morpho Blue reverts with require strings (ErrorsLib) and Solidity panics (errors.go) rather than custom errors;
// these sentinels name the strings so a refused transition says which check the real call would fail.
var (
	errMMMaxUint128Exceeded     = errors.New("everlong-flamm: morpho: max uint128 exceeded")
	errMMInsufficientLiquidity  = errors.New("everlong-flamm: morpho: insufficient liquidity")
	errMMInsufficientCollateral = errors.New("everlong-flamm: morpho: insufficient collateral")
	errMMInconsistentInput      = errors.New("everlong-flamm: morpho: inconsistent input")
	errMMZeroAssets             = errors.New("everlong-flamm: morpho: zero assets")
	errMMIrmReverted            = errors.New("everlong-flamm: morpho: irm reverted")
	errMMOracleReverted         = errors.New("everlong-flamm: morpho: oracle reverted")
)

// mmMarket is Morpho's `Market` storage struct, `market(id)`; every total is a uint128 on chain.
type mmMarket struct {
	TotalSupplyAssets uint256.Int `json:"tsa"`
	TotalSupplyShares uint256.Int `json:"tss"`
	TotalBorrowAssets uint256.Int `json:"tba"`
	TotalBorrowShares uint256.Int `json:"tbs"`
	LastUpdate        uint256.Int `json:"lastUpdate"`
	Fee               uint256.Int `json:"fee"`
}

// mmPosition is Morpho's `Position` of the financing account (`onBehalf == account`), `position(id, account)`.
type mmPosition struct {
	SupplyShares uint256.Int `json:"supplyShares"`
	BorrowShares uint256.Int `json:"borrowShares"`
	Collateral   uint256.Int `json:"collateral"`
}

// mmVenueMarket is one Morpho market as one financing account sees it: Blue's market totals and the account's
// position, the market params the account registered (`lltv`, whether an IRM is set), the AdaptiveCurveIrm's
// `rateAtTarget(id)` and the market oracle's answer. Value fields only, so a struct copy is a deep copy.
type mmVenueMarket struct {
	Market   mmMarket    `json:"market"`
	Position mmPosition  `json:"position"`
	Lltv     uint256.Int `json:"lltvWad"`
	// HasIrm is `marketParams.irm != address(0)`; Blue then accrues nothing and the account reads a zero rate.
	HasIrm bool `json:"hasIrm"`
	// IrmReadable is false while the IRM reverts (borrowRateView and borrowRate alike): the account's grace /
	// quarantine branches and Blue's accrual revert key off it.
	IrmReadable bool `json:"irmReadable"`
	// RateAtTarget is AdaptiveCurveIrm.rateAtTarget(id) (an int256 slot that is never negative).
	RateAtTarget uint256.Int `json:"rateAtTarget"`
	// OracleOk / OraclePrice are MorphoBlueAccount.oraclePrice(id): the oracle answered non-zero inside the
	// 300k gas cap. Blue's own health check reads the same price uncapped.
	OracleOk    bool        `json:"oracleOk"`
	OraclePrice uint256.Int `json:"oraclePrice"`
	// OracleZero separates the two cases oraclePrice folds into `ok == false`: the oracle answered price() == 0
	// without reverting. Blue's health check then reads that zero (and fails `insufficient collateral`) where a
	// reverting oracle bubbles its own revert. A state read through oraclePrice alone cannot tell them apart and
	// leaves it false: the transition is refused either way, only the reported revert differs.
	OracleZero bool `json:"oracleZero"`
}

// ------------------------------------------------------------------ MathLib / SharesMathLib

// mmMulDivDown is MathLib.mulDivDown: (x*y)/d, the product checked.
func mmMulDivDown(x, y, d *uint256.Int) (uint256.Int, bool) {
	var z uint256.Int
	if _, overflow := z.MulOverflow(x, y); overflow {
		return z, false
	}
	return *z.Div(&z, d), true
}

// mmMulDivUp is MathLib.mulDivUp: (x*y + (d-1))/d, the product and the sum checked.
func mmMulDivUp(x, y, d *uint256.Int) (uint256.Int, bool) {
	var z, dm1 uint256.Int
	if _, overflow := z.MulOverflow(x, y); overflow {
		return z, false
	}
	dm1.SubUint64(d, 1)
	if _, overflow := z.AddOverflow(&z, &dm1); overflow {
		return z, false
	}
	return *z.Div(&z, d), true
}

// mmWMulDown is MathLib.wMulDown.
func mmWMulDown(x, y *uint256.Int) (uint256.Int, bool) {
	return mmMulDivDown(x, y, uWad)
}

// mmWDivDown is MathLib.wDivDown.
func mmWDivDown(x, y *uint256.Int) (uint256.Int, bool) {
	return mmMulDivDown(x, uWad, y)
}

// mmWTaylorCompounded is MathLib.wTaylorCompounded: the first three terms of e^(x*n) - 1.
func mmWTaylorCompounded(x, n *uint256.Int) (uint256.Int, bool) {
	var first, sum uint256.Int
	if _, overflow := first.MulOverflow(x, n); overflow {
		return sum, false
	}
	second, ok := mmMulDivDown(&first, &first, mmTwoWad)
	if !ok {
		return sum, false
	}
	third, ok := mmMulDivDown(&second, &first, mmThreeWad)
	if !ok {
		return sum, false
	}
	if _, overflow := sum.AddOverflow(&first, &second); overflow {
		return sum, false
	}
	if _, overflow := sum.AddOverflow(&sum, &third); overflow {
		return sum, false
	}
	return sum, true
}

func mmVirtual(total, virtual *uint256.Int) (uint256.Int, bool) {
	var z uint256.Int
	_, overflow := z.AddOverflow(total, virtual)
	return z, !overflow
}

// mmToSharesDown is SharesMathLib.toSharesDown: assets*(totalShares+1e6)/(totalAssets+1).
func mmToSharesDown(assets, totalAssets, totalShares *uint256.Int) (uint256.Int, bool) {
	vs, ok1 := mmVirtual(totalShares, mmVirtualShares)
	va, ok2 := mmVirtual(totalAssets, mmVirtualAssets)
	if !ok1 || !ok2 {
		return uint256.Int{}, false
	}
	return mmMulDivDown(assets, &vs, &va)
}

// mmToAssetsDown is SharesMathLib.toAssetsDown: shares*(totalAssets+1)/(totalShares+1e6).
func mmToAssetsDown(shares, totalAssets, totalShares *uint256.Int) (uint256.Int, bool) {
	va, ok1 := mmVirtual(totalAssets, mmVirtualAssets)
	vs, ok2 := mmVirtual(totalShares, mmVirtualShares)
	if !ok1 || !ok2 {
		return uint256.Int{}, false
	}
	return mmMulDivDown(shares, &va, &vs)
}

// mmToSharesUp is SharesMathLib.toSharesUp.
func mmToSharesUp(assets, totalAssets, totalShares *uint256.Int) (uint256.Int, bool) {
	vs, ok1 := mmVirtual(totalShares, mmVirtualShares)
	va, ok2 := mmVirtual(totalAssets, mmVirtualAssets)
	if !ok1 || !ok2 {
		return uint256.Int{}, false
	}
	return mmMulDivUp(assets, &vs, &va)
}

// mmToAssetsUp is SharesMathLib.toAssetsUp.
func mmToAssetsUp(shares, totalAssets, totalShares *uint256.Int) (uint256.Int, bool) {
	va, ok1 := mmVirtual(totalAssets, mmVirtualAssets)
	vs, ok2 := mmVirtual(totalShares, mmVirtualShares)
	if !ok1 || !ok2 {
		return uint256.Int{}, false
	}
	return mmMulDivUp(shares, &va, &vs)
}

// mmAdd128 is `total += x.toUint128()`: the cast's require, then the checked uint128 add.
func mmAdd128(total *uint256.Int, x *uint256.Int) error {
	if x.Gt(mmMaxUint128) {
		return errMMMaxUint128Exceeded
	}
	var z uint256.Int
	z.Add(total, x)
	if z.Gt(mmMaxUint128) {
		return errPanicArithmetic
	}
	*total = z
	return nil
}

// mmSub128 is `total -= x.toUint128()`: the cast's require, then the checked uint128 subtraction.
func mmSub128(total *uint256.Int, x *uint256.Int) error {
	if x.Gt(mmMaxUint128) {
		return errMMMaxUint128Exceeded
	}
	if x.Gt(total) {
		return errPanicArithmetic
	}
	total.Sub(total, x)
	return nil
}

// ------------------------------------------------------------------ Morpho.sol transitions

// morphoAccrue is Morpho._accrueInterest at `now`: the IRM's (mutating) borrowRate over the stored market, the
// 3-term Taylor interest onto both borrow and supply totals, the fee shares minted at the post-interest supply
// less the fee, and lastUpdate stamped. A zero elapsed returns before the IRM is touched; a market with no IRM
// only stamps lastUpdate. The IRM runs even on a market with no borrows, so rateAtTarget still adapts.
func (v *mmVenueMarket) morphoAccrue(now uint64) error {
	m := &v.Market
	var nowU uint256.Int
	nowU.SetUint64(now)
	if nowU.Lt(&m.LastUpdate) {
		return errPanicArithmetic
	}
	var elapsed uint256.Int
	elapsed.Sub(&nowU, &m.LastUpdate)
	if elapsed.IsZero() {
		return nil
	}
	if v.HasIrm {
		if !v.IrmReadable {
			return errMMIrmReverted
		}
		rate, end, err := mmIrmBorrowRate(m, &v.RateAtTarget, now)
		if err != nil {
			return err
		}
		v.RateAtTarget = end
		growth, ok := mmWTaylorCompounded(&rate, &elapsed)
		if !ok {
			return errPanicArithmetic
		}
		interest, ok := mmWMulDown(&m.TotalBorrowAssets, &growth)
		if !ok {
			return errPanicArithmetic
		}
		if err := mmAdd128(&m.TotalBorrowAssets, &interest); err != nil {
			return err
		}
		if err := mmAdd128(&m.TotalSupplyAssets, &interest); err != nil {
			return err
		}
		if !m.Fee.IsZero() {
			feeAmount, ok := mmWMulDown(&interest, &m.Fee)
			if !ok {
				return errPanicArithmetic
			}
			var net uint256.Int
			net.Sub(&m.TotalSupplyAssets, &feeAmount)
			feeShares, ok := mmToSharesDown(&feeAmount, &net, &m.TotalSupplyShares)
			if !ok {
				return errPanicArithmetic
			}
			if err := mmAdd128(&m.TotalSupplyShares, &feeShares); err != nil {
				return err
			}
		}
	}
	m.LastUpdate = nowU
	return nil
}

// morphoIsHealthy is Morpho._isHealthy for the account: no borrow shares is healthy without an oracle read;
// otherwise toAssetsUp(borrowShares) <= wMulDown(collateral*price/1e36, lltv). A reverting oracle reverts the
// call; one answering zero is read as zero, bounding the borrow at zero (unhealthy: the borrow shares are
// non-zero, so toAssetsUp is at least one).
func (v *mmVenueMarket) morphoIsHealthy() (bool, error) {
	if v.Position.BorrowShares.IsZero() {
		return true, nil
	}
	var price uint256.Int
	switch {
	case v.OracleOk:
		price = v.OraclePrice
	case !v.OracleZero:
		return false, errMMOracleReverted
	}
	borrowed, ok := mmToAssetsUp(&v.Position.BorrowShares, &v.Market.TotalBorrowAssets, &v.Market.TotalBorrowShares)
	if !ok {
		return false, errPanicArithmetic
	}
	quoted, ok := mmMulDivDown(&v.Position.Collateral, &price, mmOraclePriceScale)
	if !ok {
		return false, errPanicArithmetic
	}
	maxBorrow, ok := mmWMulDown(&quoted, &v.Lltv)
	if !ok {
		return false, errPanicArithmetic
	}
	return !maxBorrow.Lt(&borrowed), nil
}

// morphoSupply is Morpho.supply(assets, 0): shares minted at toSharesDown after accrual.
func (v *mmVenueMarket) morphoSupply(assets *uint256.Int, now uint64) (uint256.Int, error) {
	var shares uint256.Int
	if assets.IsZero() {
		return shares, errMMInconsistentInput
	}
	if err := v.morphoAccrue(now); err != nil {
		return shares, err
	}
	m := &v.Market
	shares, ok := mmToSharesDown(assets, &m.TotalSupplyAssets, &m.TotalSupplyShares)
	if !ok {
		return shares, errPanicArithmetic
	}
	if _, overflow := new(uint256.Int).AddOverflow(&v.Position.SupplyShares, &shares); overflow {
		return shares, errPanicArithmetic
	}
	v.Position.SupplyShares.Add(&v.Position.SupplyShares, &shares)
	if err := mmAdd128(&m.TotalSupplyShares, &shares); err != nil {
		return shares, err
	}
	if err := mmAdd128(&m.TotalSupplyAssets, assets); err != nil {
		return shares, err
	}
	return shares, nil
}

// morphoWithdraw is Morpho.withdraw: exactly one of assets / shares; assets burn toSharesUp, shares pay
// toAssetsDown; the market must stay solvent (totalBorrowAssets <= totalSupplyAssets).
func (v *mmVenueMarket) morphoWithdraw(assets, shares *uint256.Int, now uint64) (uint256.Int, uint256.Int, error) {
	var outA, outS uint256.Int
	if assets.IsZero() == shares.IsZero() {
		return outA, outS, errMMInconsistentInput
	}
	if err := v.morphoAccrue(now); err != nil {
		return outA, outS, err
	}
	m := &v.Market
	var ok bool
	if !assets.IsZero() {
		outA = *assets
		if outS, ok = mmToSharesUp(assets, &m.TotalSupplyAssets, &m.TotalSupplyShares); !ok {
			return outA, outS, errPanicArithmetic
		}
	} else {
		outS = *shares
		if outA, ok = mmToAssetsDown(shares, &m.TotalSupplyAssets, &m.TotalSupplyShares); !ok {
			return outA, outS, errPanicArithmetic
		}
	}
	if outS.Gt(&v.Position.SupplyShares) {
		return outA, outS, errPanicArithmetic
	}
	v.Position.SupplyShares.Sub(&v.Position.SupplyShares, &outS)
	if err := mmSub128(&m.TotalSupplyShares, &outS); err != nil {
		return outA, outS, err
	}
	if err := mmSub128(&m.TotalSupplyAssets, &outA); err != nil {
		return outA, outS, err
	}
	if m.TotalBorrowAssets.Gt(&m.TotalSupplyAssets) {
		return outA, outS, errMMInsufficientLiquidity
	}
	return outA, outS, nil
}

// morphoBorrow is Morpho.borrow(assets, 0): shares at toSharesUp, then the health and liquidity requires.
func (v *mmVenueMarket) morphoBorrow(assets *uint256.Int, now uint64) (uint256.Int, error) {
	var shares uint256.Int
	if assets.IsZero() {
		return shares, errMMInconsistentInput
	}
	if err := v.morphoAccrue(now); err != nil {
		return shares, err
	}
	m := &v.Market
	shares, ok := mmToSharesUp(assets, &m.TotalBorrowAssets, &m.TotalBorrowShares)
	if !ok {
		return shares, errPanicArithmetic
	}
	if err := mmAdd128(&v.Position.BorrowShares, &shares); err != nil {
		return shares, err
	}
	if err := mmAdd128(&m.TotalBorrowShares, &shares); err != nil {
		return shares, err
	}
	if err := mmAdd128(&m.TotalBorrowAssets, assets); err != nil {
		return shares, err
	}
	healthy, err := v.morphoIsHealthy()
	if err != nil {
		return shares, err
	}
	if !healthy {
		return shares, errMMInsufficientCollateral
	}
	if m.TotalBorrowAssets.Gt(&m.TotalSupplyAssets) {
		return shares, errMMInsufficientLiquidity
	}
	return shares, nil
}

// morphoRepay is Morpho.repay: exactly one of assets / shares; assets burn toSharesDown, shares cost
// toAssetsUp, and the borrow total floors at zero (the repaid assets may exceed it by one wei).
func (v *mmVenueMarket) morphoRepay(assets, shares *uint256.Int, now uint64) (uint256.Int, uint256.Int, error) {
	var outA, outS uint256.Int
	if assets.IsZero() == shares.IsZero() {
		return outA, outS, errMMInconsistentInput
	}
	if err := v.morphoAccrue(now); err != nil {
		return outA, outS, err
	}
	m := &v.Market
	var ok bool
	if !assets.IsZero() {
		outA = *assets
		if outS, ok = mmToSharesDown(assets, &m.TotalBorrowAssets, &m.TotalBorrowShares); !ok {
			return outA, outS, errPanicArithmetic
		}
	} else {
		outS = *shares
		if outA, ok = mmToAssetsUp(shares, &m.TotalBorrowAssets, &m.TotalBorrowShares); !ok {
			return outA, outS, errPanicArithmetic
		}
	}
	if err := mmSub128(&v.Position.BorrowShares, &outS); err != nil {
		return outA, outS, err
	}
	if err := mmSub128(&m.TotalBorrowShares, &outS); err != nil {
		return outA, outS, err
	}
	if outA.Gt(&m.TotalBorrowAssets) {
		m.TotalBorrowAssets.Clear()
	} else {
		m.TotalBorrowAssets.Sub(&m.TotalBorrowAssets, &outA)
	}
	return outA, outS, nil
}

// morphoSupplyCollateral is Morpho.supplyCollateral: no accrual, the collateral add checked.
func (v *mmVenueMarket) morphoSupplyCollateral(assets *uint256.Int) error {
	if assets.IsZero() {
		return errMMZeroAssets
	}
	return mmAdd128(&v.Position.Collateral, assets)
}

// morphoWithdrawCollateral is Morpho.withdrawCollateral: accrual, the subtraction, then the health require.
func (v *mmVenueMarket) morphoWithdrawCollateral(assets *uint256.Int, now uint64) error {
	if assets.IsZero() {
		return errMMZeroAssets
	}
	if err := v.morphoAccrue(now); err != nil {
		return err
	}
	if err := mmSub128(&v.Position.Collateral, assets); err != nil {
		return err
	}
	healthy, err := v.morphoIsHealthy()
	if err != nil {
		return err
	}
	if !healthy {
		return errMMInsufficientCollateral
	}
	return nil
}
