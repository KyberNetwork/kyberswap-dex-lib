package smardex

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
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
func getUpdatedPriceAverage(fictiveReserveIn *uint256.Int, fictiveReserveOut *uint256.Int,
	priceAverageLastTimestamp *uint256.Int, priceAverageIn *uint256.Int, priceAverageOut *uint256.Int,
	currentTimestamp *uint256.Int) (newPriceAverageIn *uint256.Int, newPriceAverageOut *uint256.Int, err error) {

	if currentTimestamp.Cmp(priceAverageLastTimestamp) == -1 {
		err = ErrInvalidTimestamp
		return
	}

	// priceAverage is initialized with the first price at the time of the first update
	if priceAverageLastTimestamp.IsZero() || priceAverageIn.IsZero() || priceAverageOut.IsZero() {
		newPriceAverageIn = fictiveReserveIn
		newPriceAverageOut = fictiveReserveOut
	} else if priceAverageLastTimestamp.Eq(currentTimestamp) { // another tx has been done in the same timestamp
		newPriceAverageIn = priceAverageIn
		newPriceAverageOut = priceAverageOut
	} else { // need to compute new linear-average price
		// compute new price:
		timeDiff := new(uint256.Int).Sub(currentTimestamp, priceAverageLastTimestamp)
		if timeDiff.Gt(MAX_BLOCK_DIFF_SECONDS) {
			timeDiff = MAX_BLOCK_DIFF_SECONDS
		}

		newPriceAverageIn = fictiveReserveIn
		newPriceAverageOut = new(uint256.Int).Add(
			new(uint256.Int).Div(
				(new(uint256.Int).Mul(
					new(uint256.Int).Mul(
						new(uint256.Int).Sub(MAX_BLOCK_DIFF_SECONDS, timeDiff),
						priceAverageOut),
					newPriceAverageIn)),
				priceAverageIn),
			new(uint256.Int).Div(new(uint256.Int).Mul(timeDiff, fictiveReserveOut), MAX_BLOCK_DIFF_SECONDS))
	}
	return
}

func getAmountOut(param GetAmountParameters) (*GetAmountResult, error) {
	if param.reserveIn.Sign() != 1 || param.reserveOut.Sign() != 1 ||
		param.fictiveReserveIn.Sign() != 1 || param.fictiveReserveOut.Sign() != 1 {
		return nil, ErrInsufficientLiquidity
	}

	if param.priceAverageIn.Sign() != 1 || param.priceAverageOut.Sign() != 1 {
		return nil, ErrInsufficientPriceAverage
	}

	feesTotalReversed := new(uint256.Int)
	feesTotalReversed.Sub(param.feesBase, param.feesLP).Sub(feesTotalReversed, param.feesPool)

	amountInWithFees := new(uint256.Int)
	amountInWithFees.Mul(param.amount, feesTotalReversed).Div(amountInWithFees, param.feesBase)
	firstAmountIn := computeFirstTradeQtyIn(
		GetAmountParameters{
			amount:            amountInWithFees,
			reserveIn:         param.reserveIn,
			reserveOut:        param.reserveOut,
			fictiveReserveIn:  param.fictiveReserveIn,
			fictiveReserveOut: param.fictiveReserveOut,
			priceAverageIn:    param.priceAverageIn,
			priceAverageOut:   param.priceAverageOut,
			feesLP:            param.feesLP,
			feesPool:          param.feesPool,
			feesBase:          param.feesBase,
		})

	// if there is 2 trade: 1st trade mustn't re-compute fictive reserves, 2nd should
	if firstAmountIn.Eq(amountInWithFees) && ratioApproxEq(
		param.fictiveReserveIn, param.fictiveReserveOut, param.priceAverageIn, param.priceAverageOut) {
		param.fictiveReserveIn, param.fictiveReserveOut = computeFictiveReserves(
			param.reserveIn,
			param.reserveOut,
			param.fictiveReserveIn,
			param.fictiveReserveOut)
	}

	firstAmountInNoFees, _ := new(uint256.Int).MulDivOverflow(firstAmountIn, param.feesBase, feesTotalReversed)

	var (
		amountOut            *uint256.Int
		newReserveIn         *uint256.Int
		newReserveOut        *uint256.Int
		newFictiveReserveIn  *uint256.Int
		newFictiveReserveOut *uint256.Int
		err                  error
	)

	amountOut, newReserveIn, newReserveOut, newFictiveReserveIn, newFictiveReserveOut, err = applyKConstRuleOut(
		GetAmountParameters{
			amount:            firstAmountInNoFees,
			reserveIn:         param.reserveIn,
			reserveOut:        param.reserveOut,
			fictiveReserveIn:  param.fictiveReserveIn,
			fictiveReserveOut: param.fictiveReserveOut,
			priceAverageIn:    param.priceAverageIn,
			priceAverageOut:   param.priceAverageOut,
			feesLP:            param.feesLP,
			feesPool:          param.feesPool,
			feesBase:          param.feesBase,
		})
	if err != nil {
		return nil, err
	}

	// update amountIn in case there is a second trade
	param.amount = new(uint256.Int).Sub(param.amount, firstAmountInNoFees)

	// if we need a second trade
	if firstAmountIn.Lt(amountInWithFees) && firstAmountInNoFees.Lt(param.amount) {
		// in the second trade ALWAYS recompute fictive reserves
		newFictiveReserveIn, newFictiveReserveOut = computeFictiveReserves(newReserveIn, newReserveOut,
			newFictiveReserveIn, newFictiveReserveOut)

		var secondAmountOutNoFees *uint256.Int
		secondAmountOutNoFees, newReserveIn, newReserveOut,
			newFictiveReserveIn, newFictiveReserveOut, err = applyKConstRuleOut(GetAmountParameters{
			amount:            param.amount,
			reserveIn:         newReserveIn,
			reserveOut:        newReserveOut,
			fictiveReserveIn:  newFictiveReserveIn,
			fictiveReserveOut: newFictiveReserveOut,
			priceAverageIn:    param.priceAverageIn,
			priceAverageOut:   param.priceAverageOut,
			feesLP:            param.feesLP,
			feesPool:          param.feesPool,
			feesBase:          param.feesBase,
		})
		if err != nil {
			return nil, err
		}

		amountOut.Add(amountOut, secondAmountOutNoFees)
	}

	if newReserveIn.Sign() != 1 || newReserveOut.Sign() != 1 ||
		newFictiveReserveIn.Sign() != 1 || newFictiveReserveOut.Sign() != 1 {
		return nil, ErrInsufficientLiquidity
	}

	return &GetAmountResult{
		amountOut:            amountOut,
		newReserveIn:         newReserveIn,
		newReserveOut:        newReserveOut,
		newFictiveReserveIn:  newFictiveReserveIn,
		newFictiveReserveOut: newFictiveReserveOut,
	}, nil
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
func computeFictiveReserves(
	reserveIn *uint256.Int, reserveOut *uint256.Int, fictiveReserveIn *uint256.Int, fictiveReserveOut *uint256.Int,
) (newFictiveReserveIn *uint256.Int, newFictiveReserveOut *uint256.Int) {
	if new(uint256.Int).Mul(reserveOut, fictiveReserveIn).Cmp(new(uint256.Int).Mul(reserveIn, fictiveReserveOut)) < 0 {
		temp := new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(new(uint256.Int).Mul(reserveOut, reserveOut), fictiveReserveOut),
				fictiveReserveIn),
			reserveIn)
		newFictiveReserveIn = new(uint256.Int).Add(
			new(uint256.Int).Div(new(uint256.Int).Mul(temp, fictiveReserveIn), fictiveReserveOut),
			new(uint256.Int).Div(new(uint256.Int).Mul(reserveOut, fictiveReserveIn), fictiveReserveOut))
		newFictiveReserveOut = new(uint256.Int).Add(reserveOut, temp)
	} else {
		newFictiveReserveIn = new(uint256.Int).Add(
			new(uint256.Int).Div(new(uint256.Int).Mul(fictiveReserveIn, reserveOut), fictiveReserveOut),
			reserveIn)
		newFictiveReserveOut = new(uint256.Int).Add(
			new(uint256.Int).Div(new(uint256.Int).Mul(reserveIn, fictiveReserveOut), fictiveReserveIn),
			reserveOut)
	}

	// div all values by 4
	newFictiveReserveIn.Div(newFictiveReserveIn, u256.U4)
	newFictiveReserveOut.Div(newFictiveReserveOut, u256.U4)

	return

}

func computeFirstTradeQtyIn(param GetAmountParameters) *uint256.Int {
	// default value
	firstAmountIn := param.amount

	// if trade is in the good direction
	a := new(uint256.Int).Mul(param.fictiveReserveOut, param.priceAverageIn)
	b := new(uint256.Int).Mul(param.fictiveReserveIn, param.priceAverageOut)
	if a.Gt(b) {
		// pre-compute all operands
		feesTotalReversed := new(uint256.Int)
		feesTotalReversed.Sub(param.feesBase, param.feesLP).Sub(feesTotalReversed, param.feesPool)
		toSub := new(uint256.Int)
		toSub.Add(param.feesBase, feesTotalReversed).Sub(toSub, param.feesPool).Mul(toSub, param.fictiveReserveIn)
		toDiv := new(uint256.Int).Mul(new(uint256.Int).Sub(param.feesBase, param.feesPool), u256.U2)
		tmp := new(uint256.Int)
		tmp.Mul(param.fictiveReserveIn, param.fictiveReserveIn).Mul(tmp, param.feesLP).Mul(tmp, param.feesLP)
		inSqrt := new(uint256.Int)
		inSqrt.Mul(param.fictiveReserveIn, param.fictiveReserveOut).
			Mul(inSqrt, u256.U4).Div(inSqrt, param.priceAverageOut).
			Mul(inSqrt, param.priceAverageIn).Mul(inSqrt, feesTotalReversed).
			Mul(inSqrt, new(uint256.Int).Sub(param.feesBase, param.feesPool)).
			Add(inSqrt, tmp)

		// reverse sqrt check to only compute sqrt if really needed
		tmp.Mul(param.amount, toDiv).Add(tmp, toSub).Mul(tmp, tmp)
		if inSqrt.Lt(tmp) {
			firstAmountIn.Sqrt(inSqrt)
			firstAmountIn.Sub(firstAmountIn, toSub).Div(firstAmountIn, toDiv)
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
func applyKConstRuleOut(param GetAmountParameters) (amountOut *uint256.Int, newReserveIn *uint256.Int,
	newReserveOut *uint256.Int, newFictiveReserveIn *uint256.Int, newFictiveReserveOut *uint256.Int, err error) {
	// k const rule
	fee := new(uint256.Int).Sub(param.feesBase, param.feesLP)
	fee.Sub(fee, param.feesPool)

	amountInWithFee := new(uint256.Int).Mul(param.amount, fee)

	numerator := new(uint256.Int).Mul(amountInWithFee, param.fictiveReserveOut)

	denominator := new(uint256.Int).Mul(param.fictiveReserveIn, param.feesBase)
	denominator.Add(denominator, amountInWithFee)

	if denominator.IsZero() {
		err = ErrDivisionByZero
		return
	}

	amountOut = new(uint256.Int).Div(numerator, denominator)

	// update new reserves and add lp-fees to pools
	amountInWithFeeLp := new(uint256.Int).Div(new(uint256.Int).Add(amountInWithFee, new(uint256.Int).Mul(param.amount, param.feesLP)), param.feesBase)
	newReserveIn = new(uint256.Int).Add(param.reserveIn, amountInWithFeeLp)
	newFictiveReserveIn = new(uint256.Int).Add(param.fictiveReserveIn, amountInWithFeeLp)
	newReserveOut = new(uint256.Int).Sub(param.reserveOut, amountOut)
	newFictiveReserveOut = new(uint256.Int).Sub(param.fictiveReserveOut, amountOut)

	return
}
