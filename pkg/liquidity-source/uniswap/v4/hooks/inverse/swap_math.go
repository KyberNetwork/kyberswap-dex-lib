package inverse

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

const maxSwapFee = 1_000_000

var uMaxSwapFee = uint256.NewInt(maxSwapFee)

// Exact-input v4 SwapMath; shared derivation with the DualPool hook adapter.
func computeSwapStepExactIn(sqrtCurrent, sqrtTarget, liquidity, amountRemaining *uint256.Int, feePips uint64) (
	sqrtNext, amountIn, amountOut, feeAmount *uint256.Int, err error) {
	zeroForOne := sqrtCurrent.Cmp(sqrtTarget) >= 0
	fee := uint256.NewInt(feePips)
	var feeDelta uint256.Int
	feeDelta.Sub(uMaxSwapFee, fee)
	amountRemainingLessFee, err := v3Utils.MulDiv(amountRemaining, &feeDelta, uMaxSwapFee)
	if err != nil {
		return
	}
	amountIn = new(uint256.Int)
	if zeroForOne {
		err = v3Utils.GetAmount0DeltaV2(sqrtTarget, sqrtCurrent, liquidity, true, amountIn)
	} else {
		err = v3Utils.GetAmount1DeltaV2(sqrtCurrent, sqrtTarget, liquidity, true, amountIn)
	}
	if err != nil {
		return
	}
	sqrtNext = new(uint256.Int)
	feeAmount = new(uint256.Int)
	if amountRemainingLessFee.Cmp(amountIn) >= 0 {
		sqrtNext.Set(sqrtTarget)
		if feePips == maxSwapFee {
			feeAmount.Set(amountIn)
		} else {
			if feeAmount, err = v3Utils.MulDivRoundingUp(amountIn, fee, &feeDelta); err != nil {
				return
			}
		}
	} else {
		amountIn = amountRemainingLessFee
		if err = v3Utils.GetNextSqrtPriceFromInput(sqrtCurrent, liquidity, amountIn, zeroForOne, sqrtNext); err != nil {
			return
		}
		feeAmount.Sub(amountRemaining, amountIn)
	}
	amountOut = new(uint256.Int)
	if zeroForOne {
		err = v3Utils.GetAmount1DeltaV2(sqrtNext, sqrtCurrent, liquidity, false, amountOut)
	} else {
		err = v3Utils.GetAmount0DeltaV2(sqrtCurrent, sqrtNext, liquidity, false, amountOut)
	}
	return
}
