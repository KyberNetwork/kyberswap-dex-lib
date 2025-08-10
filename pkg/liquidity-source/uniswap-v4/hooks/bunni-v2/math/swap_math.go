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
	var feeAmount uint256.Int
	var temp uint256.Int

	if exactIn {
		// Calculate amountRemainingLessFee
		var amountRemainingLessFee uint256.Int
		if feePips.Eq(MAX_SWAP_FEE) {
			amountRemainingLessFee.Clear()
		} else {
			temp.Sub(MAX_SWAP_FEE, feePips)
			temp.Mul(amountRemaining, &temp)
			temp.Div(&temp, MAX_SWAP_FEE)
			amountRemainingLessFee.Set(u256.Min(&temp, SubReLU(amountRemaining, MIN_FEE_AMOUNT)))
		}

		if zeroForOne {
			amountInDelta, err := GetAmount0Delta(sqrtPriceTargetX96, sqrtPriceCurrentX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
			amountIn = amountInDelta
		} else {
			amountInDelta, err := GetAmount1Delta(sqrtPriceCurrentX96, sqrtPriceTargetX96, liquidity, true)
			if err != nil {
				return nil, nil, nil, err
			}
			amountIn = amountInDelta
		}

		if amountRemainingLessFee.Cmp(amountIn) >= 0 {
			// amountIn is capped by the target price
			sqrtPriceNextX96 = sqrtPriceTargetX96
			if feePips.Eq(MAX_SWAP_FEE) {
				feeAmount.Clear()
			} else {
				temp.Mul(amountIn, feePips)
				temp.Div(&temp, MAX_SWAP_FEE)
				temp.Sub(&temp, feePips)
				feeAmount.Set(u256.Max(&temp, MIN_FEE_AMOUNT))
			}
		} else {
			// exhaust the remaining amount
			amountIn.Set(&amountRemainingLessFee)
			sqrtPriceNextX96, err = GetNextSqrtPriceFromInput(sqrtPriceCurrentX96, liquidity, &amountRemainingLessFee, zeroForOne)
			if err != nil {
				return nil, nil, nil, err
			}
			// we didn't reach the target, so take the remainder of the maximum input as fee
			temp.Sub(amountRemaining, amountIn)
			feeAmount.Set(u256.Max(&temp, MIN_FEE_AMOUNT))
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

		// add fee back into amountIn
		// ensure that amountIn <= |amountRemaining| if exactIn
		amountIn.Add(amountIn, &feeAmount)
		if exactIn {
			amountIn.Set(u256.Min(amountIn, amountRemaining))
		}

		return sqrtPriceNextX96, amountIn, amountOut, nil
	}

	// exactOut

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
		sqrtPriceNextX96 = sqrtPriceTargetX96
	} else {
		// cap the output amount to not exceed the remaining output amount
		amountOut.Set(amountRemaining)
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

	// feePips cannot be MAX_SWAP_FEE for exact out
	temp.Mul(amountIn, feePips)
	temp.Div(&temp, MAX_SWAP_FEE)
	temp.Sub(&temp, feePips)
	feeAmount.Set(u256.Max(&temp, MIN_FEE_AMOUNT))

	// add fee back into amountIn
	amountIn.Add(amountIn, &feeAmount)

	return sqrtPriceNextX96, amountIn, amountOut, nil

}
