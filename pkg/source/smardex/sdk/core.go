package sdk

import (
	"math/big"
)

func ComputeAmountIn(
	token0 string,
	token1 string,
	reserve0 *big.Int,
	reserve1 *big.Int,
	reserve0Fic *big.Int,
	reserve1Fic *big.Int,
	tokenAmountOut *big.Int,
	tokenAddressOut string,
	priceAverageLastTimestamp int64,
	priceAverage0 *big.Int,
	priceAverage1 *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
	userTradeTimestamp int64,
	maxBlockDiffSeconds int64,
) CurrencyAmount {
	if tokenAddressOut == token0 {
		newPriceAverage1, newPriceAverage0 := getUpdatedPriceAverage(
			reserve1Fic,
			reserve0Fic,
			priceAverageLastTimestamp,
			priceAverage1,
			priceAverage0,
			userTradeTimestamp,
			maxBlockDiffSeconds,
		)

		amountIn, newRes1, newRes0, newRes1Fic, newRes0Fic :=
			getAmountIn(
				tokenAmountOut,
				reserve1,
				reserve0,
				reserve1Fic,
				reserve0Fic,
				newPriceAverage1,
				newPriceAverage0,
				feesLP,
				feesPool,
			)
		return CurrencyAmount{
			currency:         token1,
			amount:           amountIn,
			amountMax:        amountIn,
			newRes0:          newRes0,
			newRes1:          newRes1,
			newRes0Fic:       newRes0Fic,
			newRes1Fic:       newRes1Fic,
			newPriceAverage0: newPriceAverage0,
			newPriceAverage1: newPriceAverage1,
		}
	}
	newPriceAverage1, newPriceAverage0 := getUpdatedPriceAverage(
		reserve0Fic,
		reserve1Fic,
		priceAverageLastTimestamp,
		priceAverage0,
		priceAverage1,
		userTradeTimestamp,
		maxBlockDiffSeconds,
	)

	amountIn, newRes1, newRes0, newRes1Fic, newRes0Fic :=
		getAmountIn(
			tokenAmountOut,
			reserve0,
			reserve1,
			reserve0Fic,
			reserve1Fic,
			newPriceAverage0,
			newPriceAverage1,
			feesLP,
			feesPool,
		)
	return CurrencyAmount{
		currency:         token1,
		amount:           amountIn,
		amountMax:        amountIn,
		newRes0:          newRes0,
		newRes1:          newRes1,
		newRes0Fic:       newRes0Fic,
		newRes1Fic:       newRes1Fic,
		newPriceAverage0: newPriceAverage0,
		newPriceAverage1: newPriceAverage1,
	}
}

func getAmountIn(
	amountOut *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	reserveInFic *big.Int,
	reserveOutFic *big.Int,
	priceAverageIn *big.Int,
	priceAverageOut *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {
	zero := big.NewInt(0)
	if amountOut.Cmp(zero) != 1 {
		//err
	}
	if reserveIn.Cmp(zero) != 1 || reserveOut.Cmp(zero) != 1 {
		//err
	}
	if priceAverageIn.Cmp(zero) != 1 || priceAverageOut.Cmp(zero) != 1 {
		//err
	}
	reserveInFicUpdated := reserveInFic
	reserveOutFicUpdated := reserveOutFic

	firstAmount := computeFirstTradeQtyOut(amountOut, reserveInFic, reserveOutFic, priceAverageIn, priceAverageOut, feesLP, feesPool)

	if firstAmount.Cmp(amountOut) == 0 && ratioApproxEq(
		reserveInFic,
		reserveOutFic,
		priceAverageIn,
		priceAverageOut) {
		reserveInFicUpdated, reserveOutFicUpdated = computeReserveFic(
			reserveIn,
			reserveOut,
			reserveInFic,
			reserveOutFic,
		)
	}
	if reserveInFic.Cmp(amountOut) != 1 {
		return big.NewInt(0),
			reserveIn,
			reserveOut,
			reserveInFicUpdated,
			reserveOutFicUpdated
	}
	amountIn, newResIn, newResOut, newResInFic, newResOutFic :=
		applyKConstRuleIn(
			firstAmount,
			reserveIn,
			reserveOut,
			reserveInFicUpdated,
			reserveOutFicUpdated,
			feesLP,
			feesPool,
		)

	if firstAmount.Cmp(amountOut) == -1 {
		newResInFic, newResOutFic := computeReserveFic(
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
		)
		if newResInFic.Cmp(zero) != 1 {
			return big.NewInt(0),
				reserveIn,
				reserveOut,
				reserveInFicUpdated,
				reserveOutFicUpdated
		}
		sub := big.NewInt(0)
		sub.Sub(amountOut, firstAmount)
		secondAmountIn, _, _, newResInFic, newResOutFic := applyKConstRuleIn(
			sub,
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
			feesLP,
			feesPool,
		)
		amountIn.Add(amountIn, secondAmountIn)
	}
	if newResIn.Cmp(zero) != 1 ||
		newResOut.Cmp(zero) != 1 ||
		newResInFic.Cmp(zero) != 1 ||
		newResOutFic.Cmp(zero) != 1 {
		//err
	}
	return amountIn, newResIn, newResOut, newResInFic, newResOutFic
}

func applyKConstRuleIn(
	amountOut *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	reserveInFic *big.Int,
	reserveOutFic *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int) {
	feesTotalReversed := big.NewInt(0)
	feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
	numerator := big.NewInt(0)
	numerator.Mul(reserveInFic, amountOut).Mul(numerator, FEES_BASE)
	denominator := big.NewInt(0)
	denominator.Sub(reserveOutFic, amountOut).Mul(denominator, feesTotalReversed)
	if denominator.Cmp(big.NewInt(0)) == 0 {
		// err
	}
	amountIn := big.NewInt(0)
	amountIn.Div(numerator, denominator).Add(amountIn, big.NewInt(1))
	amountInWithFeeLp := big.NewInt(0)
	amountInWithFeeLp.Add(feesTotalReversed, feesLP).Mul(amountInWithFeeLp, amountIn).Div(amountInWithFeeLp, FEES_BASE)
	newResIn := big.NewInt(0)
	newResIn.Add(reserveIn, amountInWithFeeLp)
	newResInFic := big.NewInt(0)
	newResInFic.Add(reserveInFic, amountInWithFeeLp)
	newResOut := big.NewInt(0)
	newResOut.Sub(reserveOut, amountOut)
	newResOutFic := big.NewInt(0)
	newResOutFic.Sub(reserveOutFic, amountOut)

	return amountIn, newResIn, newResOut, newResInFic, newResOutFic
}

func computeReserveFic(
	reserveIn *big.Int,
	reserveOut *big.Int,
	reserveInFic *big.Int,
	reserveOutFic *big.Int,
) (*big.Int, *big.Int) {
	res1 := big.NewInt(0)
	res2 := big.NewInt(0)
	if res1.Mul(reserveOut, reserveInFic).Cmp(res2.Mul(reserveIn, reserveOutFic)) == -1 {
		temp := big.NewInt(0)
		temp.Mul(reserveOut, reserveOut).Div(temp, reserveOutFic).Mul(temp, reserveInFic).Div(temp, reserveIn)
		newResFicIn := big.NewInt(0)
		newResFicInTemp := big.NewInt(0)
		newResFicInTemp.Mul(temp, reserveInFic).Div(newResFicInTemp, reserveOutFic)
		newResFicIn.Mul(reserveOut, reserveInFic).Div(newResFicIn, reserveOutFic)
		newResFicIn.Add(newResFicIn, newResFicInTemp)
		newResFicOut := big.NewInt(0)
		newResFicOut.Add(reserveOut, temp)

		return newResFicIn.Div(newResFicIn, big.NewInt(4)), newResFicOut.Div(newResFicOut, big.NewInt(4))
	}

	newResFicIn := big.NewInt(0)
	newResFicIn.Mul(reserveInFic, reserveOut).Div(newResFicIn, reserveOutFic).Add(newResFicIn, reserveIn)
	newResFicOut := big.NewInt(0)
	newResFicOut.Mul(reserveIn, reserveOutFic).Div(newResFicOut, reserveInFic).Add(newResFicOut, reserveOut)

	return newResFicIn.Div(newResFicIn, big.NewInt(4)), newResFicOut.Div(newResFicOut, big.NewInt(4))
}

func computeFirstTradeQtyOut(
	amountOut *big.Int,
	reserveInFic *big.Int,
	reserveOutFic *big.Int,
	priceAverageIn *big.Int,
	priceAverageOut *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
) *big.Int {
	firstAmountOut := amountOut
	if new(big.Int).Mul(reserveOutFic, priceAverageIn).Cmp(new(big.Int).Mul(reserveInFic, priceAverageOut)) == 1 {

		feesTotalReversed := new(big.Int)
		feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
		reserveOutFicPredictedFees := new(big.Int)
		reserveOutFicPredictedFees.Mul(reserveInFic, feesLP).Mul(reserveOutFicPredictedFees, priceAverageOut).Div(reserveOutFicPredictedFees, priceAverageIn)
		toAdd := new(big.Int)
		toAdd.Mul(reserveOutFic, feesTotalReversed).Mul(toAdd, big.NewInt(2)).Add(toAdd, reserveOutFicPredictedFees)
		toDiv := big.NewInt(2)
		toDiv.Mul(feesTotalReversed, toDiv)
		inSqrt := new(big.Int)
		inSqrt.Sub(FEES_BASE, feesPool).
			Mul(inSqrt, feesTotalReversed).Mul(inSqrt, big.NewInt(4)).Mul(inSqrt, reserveOutFicPredictedFees).Mul(inSqrt, reserveOutFic).
			Div(inSqrt, new(big.Int).Add(feesLP, new(big.Int).Exp(reserveOutFicPredictedFees, big.NewInt(2), nil)))

		temp := new(big.Int)
		if inSqrt.Cmp(temp.Mul(amountOut, toDiv).Sub(toAdd, temp).Exp(temp, big.NewInt(2), nil)) == 1 {
			firstAmountOut = temp.Sub(toAdd, sqrt(inSqrt)).Div(temp, toDiv)
		}
	}
	return firstAmountOut
}

func getUpdatedPriceAverage(
	reserveFicIn *big.Int,
	reserveFicOut *big.Int,
	priceAverageLastTimestamp int64,
	priceAverageIn *big.Int,
	priceAverageOut *big.Int,
	currentTimestampInSecond int64,
	maxBlockDiffSeconds int64,
) (*big.Int, *big.Int) {
	zero := big.NewInt(0)

	if currentTimestampInSecond < priceAverageLastTimestamp {
		//err
	}

	if priceAverageLastTimestamp == 0 ||
		priceAverageIn.Cmp(zero) == 0 ||
		priceAverageOut.Cmp(zero) == 0 {
		return reserveFicIn, reserveFicOut
	}
	if priceAverageLastTimestamp == currentTimestampInSecond {
		return priceAverageIn, priceAverageOut
	}

	timeDiff := min(
		currentTimestampInSecond-priceAverageLastTimestamp,
		maxBlockDiffSeconds,
	)
	priceAverageInRet := reserveFicIn
	priceAverageOutRet := new(big.Int)
	priceAverageOutRet.Mul(priceAverageOut, priceAverageInRet).Mul(priceAverageOutRet, big.NewInt(maxBlockDiffSeconds-timeDiff)).Div(priceAverageOutRet, priceAverageIn)
	tmp := new(big.Int)
	tmp.Mul(reserveFicOut, big.NewInt(timeDiff)).Div(tmp, big.NewInt(maxBlockDiffSeconds))
	priceAverageOutRet.Add(priceAverageOutRet, tmp)
	return priceAverageInRet, priceAverageOutRet
}
