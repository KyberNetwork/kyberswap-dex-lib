package ambient

import "math/big"

/* @notice Correctly applies the liquidity and protocol fees to the correct side in
 *         in th pair, given how the swap is denominated. */
func assignFees(
	liqFees *big.Int, exchFees *big.Int, inBaseQty bool,
) (paidBase *big.Int, paidQuote *big.Int, paidProto *big.Int) {
	// Safe for unchecked because total fees are always previously calculated in
	// 128-bit space
	// 		 uint128 totalFees = liqFees + exchFees;
	totalFees := new(big.Int).Add(liqFees, exchFees)

	// 		 if (inBaseQty) {
	// 			 paidQuote = totalFees.toInt128Sign();
	// 		 } else {
	// 			 paidBase = totalFees.toInt128Sign();
	// 		 }
	if inBaseQty {
		paidQuote = totalFees
	} else {
		paidBase = totalFees
	}

	// 		 paidProto = exchFees;
	paidProto = exchFees

	return
}
