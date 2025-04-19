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
	result, err := div(
		new(big.Int).Lsh(afterFee, 64),
		new(big.Int).Sub(TwoPow64, new(big.Int).SetUint64(fee)),
		true,
	)
	if err != nil {
		return nil, err
	}

	if result.BitLen() > 128 {
		return nil, ErrOverflow
	}

	return result, nil
}

func computeFee(amount *big.Int, fee uint64) *big.Int {
	result, _ := MulDivOverflow(
		amount,
		new(big.Int).SetUint64(fee),
		TwoPow64,
		true,
	)

	return result
}

func noOp(sqrtRatioNext *big.Int) *SwapResult {
	return &SwapResult{
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
) (*SwapResult, error) {
	if amount.Sign() == 0 || sqrtRatio.Cmp(sqrtRatioLimit) == 0 {
		return noOp(sqrtRatio), nil
	}

	increasing := IsPriceIncreasing(amount, isToken1)

	if (sqrtRatioLimit.Cmp(sqrtRatio) == -1) == increasing {
		return nil, ErrWrongSwapDirection
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
				return &SwapResult{
					ConsumedAmount:   new(big.Int).Set(amount),
					CalculatedAmount: new(big.Int),
					SqrtRatioNext:    new(big.Int).Set(sqrtRatio),
					FeeAmount:        new(big.Int).Abs(amount),
				}, nil
			}

			var calculatedAmountExcludingFee *big.Int
			if isToken1 {
				calculatedAmountExcludingFee, err = Amount0Delta(sqrtRatioNext, sqrtRatio, liquidity, isExactOut)
			} else {
				calculatedAmountExcludingFee, err = Amount1Delta(sqrtRatioNext, sqrtRatio, liquidity, isExactOut)
			}
			if err != nil {
				return nil, err
			}

			if isExactOut {
				includingFee, err := amountBeforeFee(calculatedAmountExcludingFee, fee)
				if err != nil {
					return nil, fmt.Errorf("amount before fee: %w", err)
				}

				return &SwapResult{
					ConsumedAmount:   new(big.Int).Set(amount),
					CalculatedAmount: includingFee,
					SqrtRatioNext:    sqrtRatioNext,
					FeeAmount:        new(big.Int).Sub(includingFee, calculatedAmountExcludingFee),
				}, nil
			}

			return &SwapResult{
				ConsumedAmount:   new(big.Int).Set(amount),
				CalculatedAmount: calculatedAmountExcludingFee,
				SqrtRatioNext:    sqrtRatioNext,
				FeeAmount:        new(big.Int).Sub(new(big.Int).Abs(amount), priceImpactAmount.Abs(priceImpactAmount)),
			}, nil
		}
	}

	var specifiedAmountDelta, calculatedAmountDelta *big.Int
	if isToken1 {
		specifiedAmountDelta, err = Amount1Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactIn)
		if err != nil {
			return nil, err
		}

		calculatedAmountDelta, err = Amount0Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactOut)
		if err != nil {
			return nil, err
		}
	} else {
		specifiedAmountDelta, err = Amount0Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactIn)
		if err != nil {
			return nil, err
		}

		calculatedAmountDelta, err = Amount1Delta(sqrtRatioLimit, sqrtRatio, liquidity, isExactOut)
		if err != nil {
			return nil, err
		}
	}

	if isExactOut {
		beforeFee, err := amountBeforeFee(calculatedAmountDelta, fee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee: %w", err)
		}

		if specifiedAmountDelta.BitLen() > 127 || beforeFee.BitLen() > 128 {
			return nil, ErrOverflow
		}

		return &SwapResult{
			ConsumedAmount:   specifiedAmountDelta.Neg(specifiedAmountDelta),
			CalculatedAmount: beforeFee,
			SqrtRatioNext:    new(big.Int).Set(sqrtRatioLimit),
			FeeAmount:        calculatedAmountDelta.Sub(beforeFee, calculatedAmountDelta),
		}, nil
	} else {
		beforeFee, err := amountBeforeFee(specifiedAmountDelta, fee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee: %w", err)
		}

		if beforeFee.BitLen() > 127 || calculatedAmountDelta.BitLen() > 128 {
			return nil, ErrOverflow
		}

		return &SwapResult{
			ConsumedAmount:   beforeFee,
			CalculatedAmount: calculatedAmountDelta,
			SqrtRatioNext:    new(big.Int).Set(sqrtRatioLimit),
			FeeAmount:        specifiedAmountDelta.Sub(beforeFee, specifiedAmountDelta),
		}, nil
	}
}
