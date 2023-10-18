package sdk

import (
	"errors"
	"math/big"
)

func ComputeAmountOut(
	token0 string,
	token1 string,
	reserve0 *big.Int,
	reserve1 *big.Int,
	reserve0Fic *big.Int,
	reserve1Fic *big.Int,
	tokenAmountIn *big.Int,
	tokenAddressIn string,
	priceAverageLastTimestamp int64,
	priceAverage0 *big.Int,
	priceAverage1 *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
	userTradeTimestamp int64,
	maxBlockDiffSeconds int64,
) (*CurrencyAmount, error) {
	if tokenAddressIn == token0 {
		newPriceAverage0, newPriceAverage1, err := getUpdatedPriceAverage(
			reserve0Fic,
			reserve1Fic,
			priceAverageLastTimestamp,
			priceAverage0,
			priceAverage1,
			userTradeTimestamp,
			maxBlockDiffSeconds,
		)
		if err != nil {
			return nil, err
		}
		amountOut, newRes0, newRes1, newRes0Fic, newRes1Fic, err := getAmountOut(
			tokenAmountIn,
			reserve0,
			reserve1,
			reserve0Fic,
			reserve1Fic,
			newPriceAverage0,
			newPriceAverage1,
			feesLP,
			feesPool,
		)
		if err != nil {
			return nil, err
		}
		return &CurrencyAmount{
			currency:         token1,
			amount:           amountOut,
			amountMax:        amountOut,
			newRes0:          newRes0,
			newRes1:          newRes1,
			newRes0Fic:       newRes0Fic,
			newRes1Fic:       newRes1Fic,
			newPriceAverage0: newPriceAverage0,
			newPriceAverage1: newPriceAverage1,
		}, nil
	}
	newPriceAverage1, newPriceAverage0, err := getUpdatedPriceAverage(
		reserve1Fic,
		reserve0Fic,
		priceAverageLastTimestamp,
		priceAverage1,
		priceAverage0,
		userTradeTimestamp,
		maxBlockDiffSeconds,
	)
	if err != nil {
		return nil, err
	}

	amountOut, newRes1, newRes0, newRes1Fic, newRes0Fic, err := getAmountOut(
		tokenAmountIn,
		reserve1,
		reserve0,
		reserve1Fic,
		reserve0Fic,
		newPriceAverage1,
		newPriceAverage0,
		feesLP,
		feesPool,
	)
	if err != nil {
		return nil, err
	}
	return &CurrencyAmount{
		currency:           token0,
		amount:             amountOut,
		amountMax:          amountOut,
		newRes0:            newRes0,
		newRes1:            newRes1,
		newRes0Fic:         newRes0Fic,
		newRes1Fic:         newRes1Fic,
		newPriceAverage0:   newPriceAverage0,
		newPriceAverage1:   newPriceAverage1,
		userTradeTimestamp: userTradeTimestamp,
	}, nil
}

func getAmountOut(amountIn *big.Int, reserveIn *big.Int, reserveOut *big.Int, reserveInFic *big.Int, reserveOutFic *big.Int, priceAverageIn *big.Int, priceAverageOut *big.Int, feesLP *big.Int, feesPool *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	zero := big.NewInt(0)
	if amountIn.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_INPUT_AMOUNT")
	}
	if reserveIn.Cmp(zero) != 1 || reserveOut.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}
	if reserveInFic.Cmp(zero) != 1 || reserveOutFic.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}
	if priceAverageIn.Cmp(zero) != 1 || priceAverageOut.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_PRICE_AVERAGE")
	}
	reserveInFicUpdated := reserveInFic
	reserveOutFicUpdated := reserveOutFic

	feesTotalReversed := new(big.Int)
	feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
	amountWithFees := new(big.Int)
	amountWithFees.Mul(amountIn, feesTotalReversed).Div(amountWithFees, FEES_BASE)
	firstAmount := computeFirstTradeQtyIn(
		amountWithFees,
		reserveInFic,
		reserveOutFic,
		priceAverageIn,
		priceAverageOut,
		feesLP,
		feesPool,
	)
	if firstAmount.Cmp(amountWithFees) == 0 && ratioApproxEq(
		reserveInFic,
		reserveOutFic,
		priceAverageIn,
		priceAverageOut,
	) {
		reserveInFicUpdated, reserveOutFicUpdated = computeReserveFic(
			reserveIn,
			reserveOut,
			reserveInFic,
			reserveOutFic,
		)
	}
	if reserveInFicUpdated.Cmp(zero) != 1 {
		return zero, reserveIn, reserveOut, reserveInFicUpdated, reserveOutFicUpdated, nil
	}
	firstAmountNoFees := new(big.Int)
	firstAmountNoFees.Mul(firstAmount, FEES_BASE).Div(firstAmountNoFees, feesTotalReversed)
	amountOut, newResIn, newResOut, newResInFic, newResOutFic, err := applyKConstRuleOut(
		firstAmountNoFees,
		reserveIn,
		reserveOut,
		reserveInFicUpdated,
		reserveOutFicUpdated,
		feesLP,
		feesPool,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if firstAmount.Cmp(amountWithFees) == -1 && firstAmountNoFees.Cmp(amountIn) == -1 {
		newResInFic, newResOutFic := computeReserveFic(
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
		)
		if newResInFic.Cmp(zero) != 1 {
			return zero, reserveIn, reserveOut, reserveInFicUpdated, reserveOutFicUpdated, nil
		}
		var secondAmountOutNoFees *big.Int
		secondAmountOutNoFees, newResIn, newResOut, newResInFic, newResOutFic, err = applyKConstRuleOut(
			new(big.Int).Sub(amountIn, firstAmountNoFees),
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
			feesLP,
			feesPool,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		amountOut.Add(amountOut, secondAmountOutNoFees)
	}

	if newResIn.Cmp(zero) != 1 ||
		newResOut.Cmp(zero) != 1 ||
		newResInFic.Cmp(zero) != 1 ||
		newResOutFic.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}

	return amountOut, newResIn, newResOut, newResInFic, newResOutFic, nil
}

func applyKConstRuleOut(amountIn *big.Int, reserveIn *big.Int, reserveOut *big.Int, reserveInFic *big.Int, reserveOutFic *big.Int, feesLP *big.Int, feesPool *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	feesTotalReversed := new(big.Int)
	feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
	amountInWithFee := new(big.Int).Mul(amountIn, feesTotalReversed)
	numerator := new(big.Int).Mul(amountInWithFee, reserveOutFic)
	denominator := new(big.Int)
	denominator.Mul(reserveInFic, FEES_BASE).Sub(denominator, amountInWithFee)
	if denominator.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, nil, nil, nil, errors.New("SMARDEX_K_ERROR")
	}

	amountOut := new(big.Int).Div(numerator, denominator)

	amountInWithFeeLp := new(big.Int)
	amountInWithFeeLp.Mul(amountIn, feesLP).Add(amountInWithFeeLp, amountInWithFee).Div(amountInWithFeeLp, FEES_BASE)
	newResIn := new(big.Int).Add(reserveIn, amountInWithFeeLp)
	newResInFic := new(big.Int).Add(reserveInFic, amountInWithFeeLp)
	newResOut := new(big.Int).Sub(reserveOut, amountOut)
	newResOutFic := new(big.Int).Sub(reserveOutFic, amountOut)

	return amountOut, newResIn, newResOut, newResInFic, newResOutFic, nil
}

func computeFirstTradeQtyIn(
	amountIn *big.Int,
	reserveInFic *big.Int,
	reserveOutFic *big.Int,
	priceAverageIn *big.Int,
	priceAverageOut *big.Int,
	feesLP *big.Int,
	feesPool *big.Int,
) *big.Int {
	firstAmountIn := amountIn
	if new(big.Int).Mul(reserveOutFic, priceAverageIn).Cmp(new(big.Int).Mul(reserveInFic, priceAverageOut)) == 1 {

		feesTotalReversed := new(big.Int)
		feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
		toSub := new(big.Int)
		toSub.Add(FEES_BASE, feesTotalReversed).Sub(toSub, feesPool).Mul(toSub, reserveInFic)
		toDiv := new(big.Int)
		toDiv.Add(feesTotalReversed, feesLP).Mul(toDiv, big.NewInt(2))

		tmp := new(big.Int)
		tmp.Mul(reserveInFic, reserveInFic).Mul(tmp, feesLP).Mul(tmp, feesLP)
		inSqrt := new(big.Int)
		inSqrt.Mul(reserveInFic, reserveOutFic).Mul(inSqrt, big.NewInt(4)).Div(inSqrt, priceAverageOut).
			Mul(inSqrt, priceAverageIn).Mul(inSqrt, feesTotalReversed).
			Mul(inSqrt, new(big.Int).Sub(FEES_BASE, feesPool)).
			Add(inSqrt, tmp)

		tmp.Mul(amountIn, toDiv).Add(tmp, toSub).Exp(tmp, big.NewInt(2), nil)
		if inSqrt.Cmp(tmp) == -1 {
			firstAmountIn = sqrt(inSqrt)
			firstAmountIn.Sub(firstAmountIn, toSub).Div(firstAmountIn, toDiv)
		}
	}
	return firstAmountIn
}

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
) (*CurrencyAmount, error) {
	if tokenAddressOut == token0 {
		newPriceAverage1, newPriceAverage0, err := getUpdatedPriceAverage(
			reserve1Fic,
			reserve0Fic,
			priceAverageLastTimestamp,
			priceAverage1,
			priceAverage0,
			userTradeTimestamp,
			maxBlockDiffSeconds,
		)
		if err != nil {
			return nil, err
		}

		amountIn, newRes1, newRes0, newRes1Fic, newRes0Fic, err :=
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
		if err != nil {
			return nil, err
		}
		return &CurrencyAmount{
			currency:         token1,
			amount:           amountIn,
			amountMax:        amountIn,
			newRes0:          newRes0,
			newRes1:          newRes1,
			newRes0Fic:       newRes0Fic,
			newRes1Fic:       newRes1Fic,
			newPriceAverage0: newPriceAverage0,
			newPriceAverage1: newPriceAverage1,
		}, nil
	}
	newPriceAverage1, newPriceAverage0, err := getUpdatedPriceAverage(
		reserve0Fic,
		reserve1Fic,
		priceAverageLastTimestamp,
		priceAverage0,
		priceAverage1,
		userTradeTimestamp,
		maxBlockDiffSeconds,
	)
	if err != nil {
		return nil, err
	}

	amountIn, newRes1, newRes0, newRes1Fic, newRes0Fic, err :=
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
	if err != nil {
		return nil, err
	}
	return &CurrencyAmount{
		currency:           token1,
		amount:             amountIn,
		amountMax:          amountIn,
		newRes0:            newRes0,
		newRes1:            newRes1,
		newRes0Fic:         newRes0Fic,
		newRes1Fic:         newRes1Fic,
		newPriceAverage0:   newPriceAverage0,
		newPriceAverage1:   newPriceAverage1,
		userTradeTimestamp: userTradeTimestamp,
	}, nil
}

func getAmountIn(amountOut *big.Int, reserveIn *big.Int, reserveOut *big.Int, reserveInFic *big.Int, reserveOutFic *big.Int, priceAverageIn *big.Int, priceAverageOut *big.Int, feesLP *big.Int, feesPool *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	zero := big.NewInt(0)
	if amountOut.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	}
	if reserveIn.Cmp(zero) != 1 || reserveOut.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}
	if reserveInFic.Cmp(zero) != 1 || reserveOutFic.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}
	if priceAverageIn.Cmp(zero) != 1 || priceAverageOut.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_PRICE_AVERAGE")
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
		return big.NewInt(0), reserveIn, reserveOut, reserveInFicUpdated, reserveOutFicUpdated, nil
	}
	amountIn, newResIn, newResOut, newResInFic, newResOutFic, err :=
		applyKConstRuleIn(
			firstAmount,
			reserveIn,
			reserveOut,
			reserveInFicUpdated,
			reserveOutFicUpdated,
			feesLP,
			feesPool,
		)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if firstAmount.Cmp(amountOut) == -1 {
		newResInFic, newResOutFic := computeReserveFic(
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
		)
		if newResInFic.Cmp(zero) != 1 {
			return big.NewInt(0), reserveIn, reserveOut, reserveInFicUpdated, reserveOutFicUpdated, nil
		}
		sub := big.NewInt(0)
		sub.Sub(amountOut, firstAmount)
		var secondAmountIn *big.Int
		secondAmountIn, newResIn, newResOut, newResInFic, newResOutFic, err = applyKConstRuleIn(
			sub,
			newResIn,
			newResOut,
			newResInFic,
			newResOutFic,
			feesLP,
			feesPool,
		)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		amountIn.Add(amountIn, secondAmountIn)
	}
	if newResIn.Cmp(zero) != 1 ||
		newResOut.Cmp(zero) != 1 ||
		newResInFic.Cmp(zero) != 1 ||
		newResOutFic.Cmp(zero) != 1 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
	}
	return amountIn, newResIn, newResOut, newResInFic, newResOutFic, nil
}

func applyKConstRuleIn(amountOut *big.Int, reserveIn *big.Int, reserveOut *big.Int, reserveInFic *big.Int, reserveOutFic *big.Int, feesLP *big.Int, feesPool *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	feesTotalReversed := big.NewInt(0)
	feesTotalReversed.Sub(FEES_BASE, feesLP).Sub(feesTotalReversed, feesPool)
	numerator := big.NewInt(0)
	numerator.Mul(reserveInFic, amountOut).Mul(numerator, FEES_BASE)
	denominator := big.NewInt(0)
	denominator.Sub(reserveOutFic, amountOut).Mul(denominator, feesTotalReversed)
	if denominator.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, nil, nil, nil, errors.New("INSUFFICIENT_LIQUIDITY")
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

	return amountIn, newResIn, newResOut, newResInFic, newResOutFic, nil
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

func getUpdatedPriceAverage(reserveFicIn *big.Int, reserveFicOut *big.Int, priceAverageLastTimestamp int64, priceAverageIn *big.Int, priceAverageOut *big.Int, currentTimestampInSecond int64, maxBlockDiffSeconds int64) (*big.Int, *big.Int, error) {
	zero := big.NewInt(0)

	if currentTimestampInSecond < priceAverageLastTimestamp {
		return nil, nil, errors.New("INVALID_TIMESTAMP")
	}

	if priceAverageLastTimestamp == 0 ||
		priceAverageIn.Cmp(zero) == 0 ||
		priceAverageOut.Cmp(zero) == 0 {
		return reserveFicIn, reserveFicOut, nil
	}
	if priceAverageLastTimestamp == currentTimestampInSecond {
		return priceAverageIn, priceAverageOut, nil
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
	return priceAverageInRet, priceAverageOutRet, nil
}
