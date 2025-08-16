package math

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

	"github.com/holiman/uint256"
)

func ComputeSwapStep(
	exactIn,
	zeroForOne bool,
	sqrtPriceCurrentX96,
	sqrtPriceTargetX96,
	liquidity,
	amountRemaining,
	feePips *uint256.Int,
) (sqrtPriceNextX96, amountIn, amountOut *uint256.Int, err error) {
	var feeAmount = new(uint256.Int)
	var feeDelta = new(uint256.Int).Sub(MAX_SWAP_FEE, feePips)

	if exactIn {
		amountRemainingLessFee := u256.Min(
			MulDiv(amountRemaining, feeDelta, MAX_SWAP_FEE),
			SubReLU(amountRemaining, MIN_FEE_AMOUNT),
		)

		if zeroForOne {
			amountIn, err = GetAmount0Delta(sqrtPriceTargetX96, sqrtPriceCurrentX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountIn, err = GetAmount1Delta(sqrtPriceCurrentX96, sqrtPriceTargetX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		if amountRemainingLessFee.Cmp(amountIn) >= 0 {

			sqrtPriceNextX96 = sqrtPriceTargetX96.Clone()

			if feePips.Eq(MAX_SWAP_FEE) {
				feeAmount.Set(amountIn)
			} else {
				feeAmount.Set(u256.Max(MulDivUp(amountIn, feePips, feeDelta), MIN_FEE_AMOUNT))
			}

		} else {
			amountIn = amountRemainingLessFee.Clone()
			sqrtPriceNextX96, err = GetNextSqrtPriceFromInput(sqrtPriceCurrentX96, liquidity, amountRemainingLessFee, zeroForOne)
			if err != nil {
				return nil, nil, nil, err
			}

			feeAmount.Sub(amountRemaining, amountIn)
			feeAmount = u256.Max(feeAmount, MIN_FEE_AMOUNT)
		}

		if zeroForOne {
			amountOut, err = GetAmount1Delta(sqrtPriceNextX96, sqrtPriceCurrentX96, liquidity, false)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountOut, err = GetAmount0Delta(sqrtPriceCurrentX96, sqrtPriceNextX96, liquidity, false)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	} else {
		if zeroForOne {
			amountOut, err = GetAmount1Delta(sqrtPriceTargetX96, sqrtPriceCurrentX96, liquidity, false)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountOut, err = GetAmount0Delta(sqrtPriceCurrentX96, sqrtPriceTargetX96, liquidity, false)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		if amountRemaining.Cmp(amountOut) >= 0 {
			sqrtPriceNextX96 = sqrtPriceTargetX96.Clone()
		} else {
			amountOut = amountRemaining.Clone()
			sqrtPriceNextX96, err = GetNextSqrtPriceFromOutput(sqrtPriceCurrentX96, liquidity, amountOut, zeroForOne)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		if zeroForOne {
			amountIn, err = GetAmount0Delta(sqrtPriceNextX96, sqrtPriceCurrentX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			amountIn, err = GetAmount1Delta(sqrtPriceCurrentX96, sqrtPriceNextX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
		}

		feeAmount = u256.Max(MulDivUp(amountIn, feePips, feeDelta), MIN_FEE_AMOUNT)
	}

	if exactIn {
		var temp uint256.Int
		temp.Add(amountIn, feeAmount)
		amountIn.Set(u256.Min(&temp, amountRemaining))
	} else {
		amountIn.Add(amountIn, feeAmount)
	}

	return
}
