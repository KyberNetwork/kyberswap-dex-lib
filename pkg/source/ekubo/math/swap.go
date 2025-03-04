package math

import (
	"fmt"
	"math/big"
)

type SwapResult struct {
	ConsumedAmount   *big.Int
	CalculatedAmount *big.Int
	SqrtRatioNext    *big.Int
	FeeAmount        *big.Int
}

func IsPriceIncreasing(amount *big.Int, isToken1 bool) bool {
	return (amount.Sign() == -1) != isToken1
}

func amountBeforeFee(afterFee *big.Int, fee uint64) (*big.Int, error) {
	quotient, remainder := new(big.Int).DivMod(
		new(big.Int).Lsh(afterFee, 64),
		new(big.Int).Sub(TwoPow64, new(big.Int).SetUint64(fee)),
		new(big.Int),
	)

	if remainder.Sign() != 0 {
		quotient.Add(quotient, One)
	}

	if quotient.Cmp(TwoPow128) != -1 {
		return nil, ErrOverflow
	}

	return quotient, nil
}

func computeFee(amount *big.Int, fee uint64) *big.Int {
	num := new(big.Int).Mul(amount, new(big.Int).SetUint64(fee))
	quotient, remainder := num.DivMod(
		num,
		TwoPow64,
		new(big.Int),
	)

	if remainder.Sign() != 0 {
		quotient.Add(quotient, One)
	}

	return quotient
}

func noOp(sqrtRatioNext *big.Int) SwapResult {
	return SwapResult{
		ConsumedAmount:   new(big.Int),
		CalculatedAmount: new(big.Int),
		SqrtRatioNext:    new(big.Int).Set(sqrtRatioNext),
		FeeAmount:        new(big.Int),
	}
}

func ComputeStep(
	sqrtRatio, liquidity, sqrtRatioLimit, amount *big.Int,
	isToken1 bool,
	fee uint64,
) (SwapResult, error) {
	if amount.Sign() == 0 || sqrtRatio.Cmp(sqrtRatioLimit) == 0 {
		return noOp(sqrtRatio), nil
	}

	increasing := IsPriceIncreasing(amount, isToken1)

	if (sqrtRatioLimit.Cmp(sqrtRatio) == -1) == increasing {
		return SwapResult{}, ErrWrongSwapDirection
	}

	if liquidity.Sign() == 0 {
		return noOp(sqrtRatioLimit), nil
	}

	isExactIn := amount.Sign() == 1
	isExactOut := amount.Sign() == -1

	var priceImpactAmount *big.Int
	if isExactOut {
		priceImpactAmount = new(big.Int).Set(amount)
	} else {
		fee := computeFee(amount, fee)
		priceImpactAmount = fee.Sub(amount, fee)
	}

	var (
		sqrtRatioNext *big.Int
		err           error
	)
	if isToken1 {
		sqrtRatioNext, err = nextSqrtRatioFromAmount1(sqrtRatio, liquidity, priceImpactAmount)
	} else {
		sqrtRatioNext, err = nextSqrtRatioFromAmount0(sqrtRatio, liquidity, priceImpactAmount)
	}

	if err == nil {
		if (sqrtRatioNext.Cmp(sqrtRatioLimit) != 1) == increasing {
			if sqrtRatioNext.Cmp(sqrtRatio) == 0 {
				return SwapResult{
					ConsumedAmount:   new(big.Int).Set(amount),
					CalculatedAmount: new(big.Int),
					SqrtRatioNext:    new(big.Int).Set(sqrtRatio),
					FeeAmount:        new(big.Int).Abs(amount),
				}, nil
			}

			var calculatedAmountExcludingFee *big.Int
			if isToken1 {
				calculatedAmountExcludingFee, err = amount0Delta(sqrtRatioNext, sqrtRatio, liquidity, isExactOut)
			} else {
				calculatedAmountExcludingFee, err = amount1Delta(sqrtRatioNext, sqrtRatio, liquidity, isExactOut)
			}

			if err != nil {
				return SwapResult{}, fmt.Errorf("amount delta: %w", err)
			}

			if isExactOut {
				includingFee, err := amountBeforeFee(calculatedAmountExcludingFee, fee)
				if err != nil {
					return SwapResult{}, fmt.Errorf("amount before fee: %w", err)
				}

				return SwapResult{
					ConsumedAmount:   new(big.Int).Set(amount),
					CalculatedAmount: includingFee,
					SqrtRatioNext:    sqrtRatioNext,
					FeeAmount:        includingFee.Sub(includingFee, calculatedAmountExcludingFee),
				}, nil
			}

			return SwapResult{
				ConsumedAmount:   new(big.Int).Set(amount),
				CalculatedAmount: calculatedAmountExcludingFee,
				SqrtRatioNext:    sqrtRatioNext,
				FeeAmount:        new(big.Int).Sub(new(big.Int).Abs(amount), priceImpactAmount.Abs(priceImpactAmount)),
			}, nil
		}
	}

	var specifiedAmountDelta, calculatedAmountDelta *big.Int
	if isToken1 {
		specifiedAmountDelta, err = amount1Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactIn)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount1 delta: %w", err)
		}

		calculatedAmountDelta, err = amount0Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactOut)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount0 delta: %w", err)
		}
	} else {
		specifiedAmountDelta, err = amount0Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactIn)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount1 delta: %w", err)
		}

		calculatedAmountDelta, err = amount0Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactOut)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount0 delta: %w", err)
		}
	}

	if isExactOut {
		beforeFee, err := amountBeforeFee(calculatedAmountDelta, fee)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount before fee: %w", err)
		}

		if specifiedAmountDelta.Cmp(TwoPow127) != -1 || beforeFee.Cmp(TwoPow127) != -1 {
			return SwapResult{}, ErrOverflow
		}

		return SwapResult{
			ConsumedAmount:   specifiedAmountDelta.Neg(specifiedAmountDelta),
			CalculatedAmount: beforeFee,
			SqrtRatioNext:    new(big.Int).Set(sqrtRatioLimit),
			FeeAmount:        calculatedAmountDelta.Sub(beforeFee, calculatedAmountDelta),
		}, nil
	} else {
		beforeFee, err := amountBeforeFee(specifiedAmountDelta, fee)
		if err != nil {
			return SwapResult{}, fmt.Errorf("amount before fee: %w", err)
		}

		if beforeFee.Cmp(TwoPow127) != -1 || calculatedAmountDelta.Cmp(TwoPow127) != -1 {
			return SwapResult{}, ErrOverflow
		}

		return SwapResult{
			ConsumedAmount:   beforeFee,
			CalculatedAmount: calculatedAmountDelta,
			SqrtRatioNext:    new(big.Int).Set(sqrtRatioLimit),
			FeeAmount:        specifiedAmountDelta.Sub(beforeFee, specifiedAmountDelta),
		}, nil
	}
}
