package math

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type SwapResult struct {
	ConsumedAmount   *uint256.Int
	CalculatedAmount *uint256.Int
	SqrtRatioNext    *uint256.Int
	FeeAmount        *uint256.Int
}

func IsPriceIncreasing(amount *uint256.Int, isToken1 bool) bool {
	return (amount.Sign() < 0) != isToken1
}

func AmountBeforeFee(afterFee *uint256.Int, fee uint64) (*uint256.Int, error) {
	var tmp, tmp2 uint256.Int
	result, err := div(
		tmp.Lsh(afterFee, 64),
		tmp2.SubUint64(big256.U2Pow64, fee),
		true,
	)
	if err != nil {
		return nil, err
	} else if result.BitLen() > 128 {
		return nil, ErrOverflow
	}

	return result, nil
}

func ComputeFee(amount *uint256.Int, fee uint64) *uint256.Int {
	result, _ := MulDivOverflow(
		amount,
		new(uint256.Int).SetUint64(fee),
		big256.U2Pow64,
		true,
	)

	return result
}

func noOp(sqrtRatioNext *uint256.Int) *SwapResult {
	return &SwapResult{
		ConsumedAmount:   new(uint256.Int),
		CalculatedAmount: new(uint256.Int),
		SqrtRatioNext:    new(uint256.Int).Set(sqrtRatioNext),
		FeeAmount:        new(uint256.Int),
	}
}

func ComputeStep(
	sqrtRatio, liquidity, sqrtRatioLimit, amount *uint256.Int,
	isToken1 bool,
	fee uint64,
) (*SwapResult, error) {
	if amount.IsZero() || sqrtRatio.Eq(sqrtRatioLimit) {
		return noOp(sqrtRatio), nil
	}

	increasing := IsPriceIncreasing(amount, isToken1)
	if sqrtRatioLimit.Lt(sqrtRatio) == increasing {
		return nil, ErrWrongSwapDirection
	} else if liquidity.IsZero() {
		return noOp(sqrtRatioLimit), nil
	}

	isExactIn, isExactOut := amount.Sign() > 0, amount.Sign() < 0
	var priceImpactAmount *uint256.Int
	if isExactOut {
		priceImpactAmount = amount.Clone()
	} else {
		fee := ComputeFee(amount, fee)
		priceImpactAmount = fee.Sub(amount, fee)
	}

	sqrtRatioNext, err := lo.Ternary(isToken1, nextSqrtRatioFromAmount1, nextSqrtRatioFromAmount0)(
		sqrtRatio, liquidity, priceImpactAmount)
	if err == nil {
		if sqrtRatioNext.Gt(sqrtRatioLimit) != increasing {
			if sqrtRatioNext.Eq(sqrtRatio) {
				return &SwapResult{
					ConsumedAmount:   new(uint256.Int).Set(amount),
					CalculatedAmount: new(uint256.Int),
					SqrtRatioNext:    new(uint256.Int).Set(sqrtRatio),
					FeeAmount:        new(uint256.Int).Abs(amount),
				}, nil
			}

			calculatedAmountExcludingFee, err := lo.Ternary(isToken1, Amount0Delta, Amount1Delta)(
				sqrtRatioNext, sqrtRatio, liquidity, isExactOut)
			if err != nil {
				return nil, err
			}

			if isExactOut {
				includingFee, err := AmountBeforeFee(calculatedAmountExcludingFee, fee)
				if err != nil {
					return nil, fmt.Errorf("amount before fee: %w", err)
				}

				return &SwapResult{
					ConsumedAmount:   new(uint256.Int).Set(amount),
					CalculatedAmount: includingFee,
					SqrtRatioNext:    sqrtRatioNext,
					FeeAmount:        new(uint256.Int).Sub(includingFee, calculatedAmountExcludingFee),
				}, nil
			}

			return &SwapResult{
				ConsumedAmount:   new(uint256.Int).Set(amount),
				CalculatedAmount: calculatedAmountExcludingFee,
				SqrtRatioNext:    sqrtRatioNext,
				FeeAmount: new(uint256.Int).Sub(new(uint256.Int).Abs(amount),
					priceImpactAmount.Abs(priceImpactAmount)),
			}, nil
		}
	}

	var specifiedAmountDelta, calculatedAmountDelta *uint256.Int
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
		beforeFee, err := AmountBeforeFee(calculatedAmountDelta, fee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee: %w", err)
		}

		if specifiedAmountDelta.BitLen() > 127 || beforeFee.BitLen() > 128 {
			return nil, ErrOverflow
		}

		return &SwapResult{
			ConsumedAmount:   specifiedAmountDelta.Neg(specifiedAmountDelta),
			CalculatedAmount: beforeFee,
			SqrtRatioNext:    new(uint256.Int).Set(sqrtRatioLimit),
			FeeAmount:        calculatedAmountDelta.Sub(beforeFee, calculatedAmountDelta),
		}, nil
	} else {
		beforeFee, err := AmountBeforeFee(specifiedAmountDelta, fee)
		if err != nil {
			return nil, fmt.Errorf("amount before fee: %w", err)
		}

		if beforeFee.BitLen() > 127 || calculatedAmountDelta.BitLen() > 128 {
			return nil, ErrOverflow
		}

		return &SwapResult{
			ConsumedAmount:   beforeFee,
			CalculatedAmount: calculatedAmountDelta,
			SqrtRatioNext:    new(uint256.Int).Set(sqrtRatioLimit),
			FeeAmount:        specifiedAmountDelta.Sub(beforeFee, specifiedAmountDelta),
		}, nil
	}
}
