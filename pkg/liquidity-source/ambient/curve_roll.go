package ambient

import (
	"errors"
	"math/big"
)

/* @notice Called when a curve has reached its a  bump barrier. Because the
 *   barrier occurs at the final price in the tick, we need to "shave the price"
 *   over into the next tick. The curve has kicked in liquidity that's only active
 *   below this price, and we need the price to reflect the correct tick. So we burn
 *   an economically meaningless amount of collateral token wei to shift the price
 *   down by exactly one unit of precision into the next tick. */
func shaveAtBump(curve *curveState, inBaseQty bool, isBuy bool, swapLeft *big.Int) (*big.Int, *big.Int, *big.Int, error) {
	// uint128 burnDown = CurveMath.priceToTokenPrecision
	// (curve.activeLiquidity(), curve.priceRoot_, inBaseQty);
	actLiq, err := curve.activeLiquidity()
	if err != nil {
		return nil, nil, nil, err
	}
	burnDown := priceToTokenPrecision(actLiq, curve.priceRoot, inBaseQty)

	// require(swapLeft > burnDown, "BD");
	if swapLeft.Cmp(burnDown) < 1 {
		return nil, nil, nil, errors.New("BD")
	}

	// if (isBuy) {
	// return setShaveUp(curve, inBaseQty, burnDown);
	// }
	if isBuy {
		paidBase, paidQuote, burnSwap := setShaveUp(curve, inBaseQty, burnDown)
		return paidBase, paidQuote, burnSwap, nil
	}

	// return setShaveDown(curve, inBaseQty, burnDown);
	paidBase, paidQuote, burnSwap := setShaveDown(curve, inBaseQty, burnDown)
	return paidBase, paidQuote, burnSwap, nil
}

/* @notice After calculating a burn down amount of collateral, roll the curve over
*         into the next tick above the current tick. */
func setShaveUp(curve *curveState, inBaseQty bool, burnDown *big.Int) (*big.Int, *big.Int, *big.Int) {
	// if (curve.priceRoot_ < TickMath.MAX_SQRT_RATIO - 1) {
	// curve.priceRoot_ += 1; // MAX_SQRT is well below uint128.max
	// }
	tmpMaxSQRTRatio := new(big.Int).Set(bigMaxSQRTRatio)
	tmpMaxSQRTRatio.Sub(tmpMaxSQRTRatio, big1)
	if curve.priceRoot.Cmp(tmpMaxSQRTRatio) == -1 {
		curve.priceRoot.Add(curve.priceRoot, big1)
	}

	// // When moving the price up at constant liquidity, no additional quote tokens are required for
	// // collateralization
	// paidQuote = 0;
	paidQuote := new(big.Int).Set(big0)

	// When moving the price up at constant liquidity, the swapper must pay a small amount of additional
	// base tokens to keep the curve over-collateralized.
	// paidBase = burnDown.toInt128Sign();
	paidBase := new(big.Int).Set(burnDown)

	// // If the fixed swap leg is in quote tokens, then this has zero impact, if the swap leg is in base
	// // tokens then we have to adjust the deduct the quote tokens the user paid above from the remaining swap
	// // quantity
	// burnSwap = inBaseQty ? burnDown : 0;
	burnSwap := new(big.Int)
	if inBaseQty {
		burnSwap.Set(burnDown)
	} else {
		burnSwap.Set(big0)
	}

	return paidBase, paidQuote, burnSwap
}

/* @notice After calculating a burn down amount of collateral, roll the curve over
*         into the next tick below the current tick.
*
* @dev    This is used to handle the situation when we've reached the end of a liquidity
*         range, and need to safely move the curve by one price unit to move it over into
*         the next liquidity range. Although a single price unit is almost always economically
*         de minims, there are small flows needed to move the curve price while remaining safely
*         over-collateralized.
*
* @param curve The liquidity curve, which will be adjusted to move the price one unit.
* @param inBaseQty If true indicates that the swap is made with fixed base tokens and floating quote
*                  tokens.
* @param burnDown The pre-calculated amount of tokens needed to maintain over-collateralization when
*                 moving the curve by one price unit.
*
* @return paidBase The additional amount of base tokens that the swapper should pay to the curve to
*                  move the price one unit.
* @return paidQuote The additional amount of quote tokens the swapper should pay to the curve.
* @return burnSwap  The amount of tokens to remove from the remaining fixed leg of the swap quantity. */
func setShaveDown(curve *curveState, inBaseQty bool, burnDown *big.Int) (*big.Int, *big.Int, *big.Int) {
	// if (curve.priceRoot_ > TickMath.MIN_SQRT_RATIO) {
	// curve.priceRoot_ -= 1; // MIN_SQRT is well above uint128 0
	// }
	if curve.priceRoot.Cmp(bigMinSQRTRatio) == 1 {
		curve.priceRoot.Sub(curve.priceRoot, big1)
	}

	// // When moving the price down at constant liquidity, no additional base tokens are required for
	// // collateralization
	// paidBase = 0;
	paidBase := big.NewInt(0)

	// // When moving the price down at constant liquidity, the swapper must pay a small amount of additional
	// // quote tokens to keep the curve over-collateralized.
	// paidQuote = burnDown.toInt128Sign();
	paidQuote := new(big.Int).Set(burnDown)

	// // If the fixed swap leg is in base tokens, then this has zero impact, if the swap leg is in quote
	// // tokens then we have to adjust the deduct the quote tokens the user paid above from the remaining swap
	// // quantity
	// burnSwap = inBaseQty ? 0 : burnDown;
	var burnSwap = new(big.Int)
	if inBaseQty {
		burnSwap.Set(big0)
	} else {
		burnSwap.Set(burnDown)
	}

	return paidBase, paidQuote, burnSwap
}
