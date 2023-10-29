package smardex

import (
	"math/big"
)

/**
 * If the reserves in the SC are imbalanced and the SC seeks to sell one of the tokens,
 * it leaves an opportunity for a user to make a swap in one direction and then in the opposite direction.
 * User would sell at a higher price than they bought, resulting in receiving more tokens than they spent.
 * This technique called price variation mechanism, it introduces Price Average which is a price that varies based on a 300-second moving average
 * If the user tries to make a swap in one direction and then in the opposite direction,
 * they will first make a trade that will significantly unbalance the pair,
 * and the fictive reserve will be recalculated because at the beginning, the price and the priceAverage are the same.
 * The user will then make the opposite trade in the same block, but the priceAverage will not have been updated yet
 * so the k constant rule will be applied to return to the price
 **/
func getUpdatedPriceAverage(fictiveReserveIn *big.Int, fictiveReserveOut *big.Int,
	priceAverageLastTimestamp int64, priceAverageIn *big.Int,
	priceAverageOut *big.Int, currentTimestamp int64) (newPriceAverageIn *big.Int, newPriceAverageOut *big.Int, err error) {

	if currentTimestamp >= 0 {
		err = ErrInvalidTimestamp
		return
	}

	// priceAverage is initialized with the first price at the time of the first update
	if priceAverageLastTimestamp == 0 {
		newPriceAverageIn = fictiveReserveIn
		newPriceAverageOut = fictiveReserveOut
	} else if priceAverageLastTimestamp == currentTimestamp { // another tx has been done in the same timestamp
		newPriceAverageIn = priceAverageIn
		newPriceAverageOut = priceAverageOut
	} else { // need to compute new linear-average price
		// compute new price:
		timeDiff := min(currentTimestamp-priceAverageLastTimestamp, MAX_BLOCK_DIFF_SECONDS)

		newPriceAverageIn = fictiveReserveIn
		newPriceAverageOut = new(big.Int).Div(
			new(big.Int).Add(
				new(big.Int).Div(
					(new(big.Int).Mul(
						new(big.Int).Mul(
							big.NewInt(MAX_BLOCK_DIFF_SECONDS-timeDiff),
							priceAverageOut),
						newPriceAverageIn)),
					priceAverageIn),
				new(big.Int).Mul(big.NewInt(timeDiff), fictiveReserveOut)),
			big.NewInt(MAX_BLOCK_DIFF_SECONDS))
	}
	return
}

func getAmountOut(param GetAmountParameters) (GetAmountResult, error) {
	result := GetAmountResult{}

	if isZero(param.reserveIn) || isZero(param.reserveOut) ||
		isZero(param.fictiveReserveIn) || isZero(param.fictiveReserveOut) {
		return result, ErrInsufficientLiquidity
	}

	amountInWithFees := new(big.Int).Div(
		new(big.Int).Mul(
			param.amount,
			new(big.Int).Sub(new(big.Int).Sub(FEES_BASE, param.feesPool), param.feesLP)),
		FEES_BASE)
	firstAmountIn := computeFirstTradeQtyIn(param)

	// if there is 2 trade: 1st trade mustn't re-compute fictive reserves, 2nd should
	if firstAmountIn.Cmp(amountInWithFees) == 0 && ratioApproxEq(
		param.fictiveReserveIn, param.fictiveReserveOut, param.priceAverageIn, param.priceAverageOut) {
		param.fictiveReserveIn, param.fictiveReserveOut = computeFictiveReserves(
			param.reserveIn,
			param.reserveOut,
			param.fictiveReserveIn,
			param.fictiveReserveOut)
	}

	firstAmountInNoFees := new(big.Int).Div(
		new(big.Int).Mul(firstAmountIn, FEES_BASE),
		new(big.Int).Sub(new(big.Int).Sub(FEES_BASE, param.feesPool), param.feesLP))

	result.amountOut, result.newReserveIn, result.newReserveOut,
		result.newFictiveReserveIn, result.newFictiveReserveOut = applyKConstRuleOut(
		GetAmountParameters{
			amount:            firstAmountInNoFees,
			reserveIn:         param.reserveIn,
			reserveOut:        param.reserveOut,
			fictiveReserveIn:  param.fictiveReserveIn,
			fictiveReserveOut: param.fictiveReserveOut,
			priceAverageIn:    param.priceAverageIn,
			priceAverageOut:   param.priceAverageOut,
			feesLP:            param.feesLP,
			feesPool:          param.feesPool})

	// update amountIn in case there is a second trade
	param.amount = new(big.Int).Sub(param.amount, firstAmountInNoFees)

	// if we need a second trade
	if firstAmountIn.Cmp(amountInWithFees) < 0 {
		// in the second trade ALWAYS recompute fictive reserves
		result.newFictiveReserveIn, result.newFictiveReserveOut = computeFictiveReserves(
			result.newReserveIn,
			result.newReserveOut,
			result.newFictiveReserveIn,
			result.newFictiveReserveOut)

		var secondAmountOutNoFees *big.Int
		secondAmountOutNoFees, result.newReserveIn, result.newReserveOut,
			result.newFictiveReserveIn, result.newFictiveReserveOut = applyKConstRuleOut(GetAmountParameters{
			amount:            param.amount,
			reserveIn:         result.newReserveIn,
			reserveOut:        result.newReserveOut,
			fictiveReserveIn:  result.newFictiveReserveIn,
			fictiveReserveOut: result.newFictiveReserveOut,
			priceAverageIn:    param.priceAverageIn,
			priceAverageOut:   param.priceAverageOut,
			feesLP:            param.feesLP,
			feesPool:          param.feesPool})

		result.amountOut = new(big.Int).Add(result.amountOut, secondAmountOutNoFees)
	}

	return result, nil
}

/**
* @notice compute fictive reserves
* @param reserveIn reserve of the in-token
* @param reserveOut reserve of the out-token
* @param fictiveReserveIn fictive reserve of the in-token
* @param fictiveReserveOut fictive reserve of the out-token
* @return newFictiveReserveIn new fictive reserve of the in-token
* @return newFictiveReserveOut new fictive reserve of the out-token
 */
func computeFictiveReserves(reserveIn *big.Int, reserveOut *big.Int, fictiveReserveIn *big.Int, fictiveReserveOut *big.Int) (newFictiveReserveIn *big.Int, newFictiveReserveOut *big.Int) {
	if new(big.Int).Mul(reserveOut, fictiveReserveIn).Cmp(new(big.Int).Mul(reserveIn, fictiveReserveOut)) < 0 {
		temp := new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(reserveOut, reserveOut), fictiveReserveOut),
				fictiveReserveIn),
			reserveIn)
		newFictiveReserveIn = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(temp, fictiveReserveIn), fictiveReserveOut),
			new(big.Int).Div(new(big.Int).Mul(reserveOut, fictiveReserveIn), fictiveReserveOut))
		newFictiveReserveOut = new(big.Int).Add(reserveOut, temp)
	} else {
		newFictiveReserveIn = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(fictiveReserveIn, reserveOut), fictiveReserveOut),
			reserveIn)
		newFictiveReserveOut = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(reserveIn, fictiveReserveOut), fictiveReserveIn),
			reserveOut)
	}

	// div all values by 4
	newFictiveReserveIn = new(big.Int).Div(newFictiveReserveIn, big.NewInt(4))
	newFictiveReserveOut = new(big.Int).Div(newFictiveReserveOut, big.NewInt(4))
	return

}

func computeFirstTradeQtyIn(param GetAmountParameters) *big.Int {
	// default value
	firstAmountIn := param.amount

	// if trade is in the good direction
	a := new(big.Int).Mul(param.fictiveReserveOut, param.priceAverageIn)
	b := new(big.Int).Mul(param.fictiveReserveIn, param.priceAverageOut)
	if a.Cmp(b) == 1 {
		// pre-compute all operands
		toSub := new(big.Int).Mul(
			param.fictiveReserveIn,
			new(big.Int).Sub(
				new(big.Int).Sub(
					new(big.Int).Mul(FEES_BASE, big.NewInt(2)),
					new(big.Int).Mul(param.feesPool, big.NewInt(2))),
				param.feesLP))
		toDiv := new(big.Int).Mul(new(big.Int).Sub(FEES_BASE, param.feesPool), big.NewInt(2))
		inSqrt := new(big.Int).Mul(
			new(big.Int).Div(
				new(big.Int).Mul(new(big.Int).Mul(param.fictiveReserveIn, param.fictiveReserveOut), big.NewInt(4)),
				param.priceAverageOut),
			new(big.Int).Add(
				new(big.Int).Mul(
					param.priceAverageIn,
					new(big.Int).Mul(
						new(big.Int).Sub(new(big.Int).Sub(FEES_BASE, param.feesLP), param.feesPool),
						new(big.Int).Sub(FEES_BASE, param.feesPool))),
				new(big.Int).Mul(
					new(big.Int).Mul(param.fictiveReserveIn, param.fictiveReserveIn),
					new(big.Int).Mul(param.feesLP, param.feesLP))))

		// reverse sqrt check to only compute sqrt if really needed
		inSqrtCompare := new(big.Int).Add(toSub, new(big.Int).Mul(param.amount, toDiv))
		if inSqrt.Cmp(new(big.Int).Mul(inSqrtCompare, inSqrtCompare)) < 0 {
			firstAmountIn = new(big.Int).Div(new(big.Int).Sub(new(big.Int).Sqrt(inSqrt), toSub), toDiv)
		}
	}

	return firstAmountIn
}

/**
* @notice apply k const rule using fictive reserve, when the amountIn is specified
* @param param contain all params required from struct GetAmountParameters
* @return amountOut qty of token that leaves in the contract
* @return newReserveIn new reserve of the in-token after the transaction
* @return newReserveOut new reserve of the out-token after the transaction
* @return newFictiveReserveIn new fictive reserve of the in-token after the transaction
* @return newFictiveReserveOut new fictive reserve of the out-token after the transaction
 */
func applyKConstRuleOut(param GetAmountParameters) (amountOut *big.Int, newReserveIn *big.Int, newReserveOut *big.Int, newFictiveReserveIn *big.Int, newFictiveReserveOut *big.Int) {
	// k const rule
	amountInWithFee := new(big.Int).Mul(param.amount, new(big.Int).Sub(new(big.Int).Sub(FEES_BASE, param.feesLP), param.feesPool))
	numerator := new(big.Int).Mul(amountInWithFee, param.fictiveReserveOut)
	denominator := new(big.Int).Add(new(big.Int).Mul(param.fictiveReserveIn, FEES_BASE), amountInWithFee)
	amountOut = new(big.Int).Div(numerator, denominator)

	// update new reserves and add lp-fees to pools
	amountInWithFeeLp := new(big.Int).Div(new(big.Int).Add(amountInWithFee, new(big.Int).Mul(param.amount, param.feesLP)), FEES_BASE)
	newReserveIn = new(big.Int).Add(param.reserveIn, amountInWithFeeLp)
	newFictiveReserveIn = new(big.Int).Add(param.fictiveReserveIn, amountInWithFeeLp)
	newReserveOut = new(big.Int).Sub(param.reserveOut, amountOut)
	newFictiveReserveOut = new(big.Int).Sub(param.fictiveReserveOut, amountOut)
	return
}
