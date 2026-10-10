package lotflow

import "math/big"

// Basket mint/redeem rounding (MemeFiLP contracts/src/Basket.sol). Units are wei of each constituent
// per UnitBase (1e18) LOT. The asymmetry is deliberate on chain and must be kept here:
//   - mint (amountsFor):  ceil(unit*x/1e18) — a buy acquires at least the backing;
//   - redeem (_redeem):   floor(unit*x/1e18) — a sell never receives more than the backing.

// UnitBase is Basket.UNIT.
var UnitBase = bigE18

// MintAmounts is Basket.amountsFor(lotAmt): the constituent amounts a mint of lotAmt LOT consumes.
func MintAmounts(units []*big.Int, lotAmt *big.Int) []*big.Int {
	out := make([]*big.Int, len(units))
	for i, u := range units {
		out[i] = mulDivUp(u, lotAmt, UnitBase)
	}
	return out
}

// RedeemAmounts is what Basket.redeem(lotAmt) pays out of each constituent.
func RedeemAmounts(units []*big.Int, lotAmt *big.Int) []*big.Int {
	out := make([]*big.Int, len(units))
	for i, u := range units {
		out[i] = mulDiv(u, lotAmt, UnitBase)
	}
	return out
}
