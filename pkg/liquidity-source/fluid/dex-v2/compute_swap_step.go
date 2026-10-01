package dexv2

import (
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/samber/lo"
)

func computeSwapStepForSwapInWithoutFee(
	sqrtRatioCurrentX96,
	sqrtRatioTargetX96 *utils.Uint160,
	liquidity *utils.Uint128,
	amountRemaining *utils.Int256,

	sqrtRatioNextX96 *utils.Uint160, amountIn, amountOut, feeAmount *utils.Uint256,
) error {
	return utils.ComputeSwapStep(sqrtRatioCurrentX96, sqrtRatioTargetX96, liquidity, amountRemaining,
		0, // set fee to 0
		sqrtRatioNextX96, amountIn, amountOut, feeAmount)
}

func computeSwapStepForSwapInWithDynamicFee(
	zeroToOne bool,
	sqrtRatioCurrentX96,
	sqrtRatioTargetX96 *utils.Uint160,
	liquidity *utils.Uint128,
	amountRemaining *utils.Int256,

	d DynamicFeeVariablesUI,
	protocolFee *utils.Uint128,

	sqrtPriceNextX96 *utils.Uint160, amountIn, amountOut, feeAmount, protocolFeeAmount, lpFeeAmount *utils.Uint256,
) error {
	sqrtPriceNextX96.Set(sqrtRatioCurrentX96)
	var priceNextX96 utils.Uint256
	err := utils.MulDivV2(sqrtPriceNextX96, sqrtPriceNextX96, Q96I, &priceNextX96, nil)
	if err != nil {
		return err
	}

	if lo.Ternary(
		zeroToOne,
		priceNextX96.Cmp(d.minFeeKinkPriceX96) > 0,
		priceNextX96.Cmp(d.minFeeKinkPriceX96) < 0,
	) {
		err := computeSwapStepForSwapInWithoutFee(
			sqrtRatioCurrentX96,
			calculateStepTargetSqrtPriceX96(zeroToOne, d.minFeeKinkPriceX96, sqrtRatioTargetX96),
			liquidity,
			amountRemaining,
			sqrtPriceNextX96,
			amountIn,
			amountOut,
			feeAmount,
		)
		if err != nil {
			return err
		}

		var amountInSigned utils.Int256
		err = utils.ToInt256(amountIn, &amountInSigned)
		if err != nil {
			return err
		}

		amountRemaining.Sub(amountRemaining, &amountInSigned)
		if amountOut.Cmp(X86UI) > 0 {
			return ErrGreaterThanMaxAmountOut
		}

		var stepProtocolFee utils.Uint256
		stepProtocolFee.Mul(amountOut, protocolFee).
			Div(&stepProtocolFee, SIX_DECIMALS_UI)
		protocolFeeAmount.Add(protocolFeeAmount, &stepProtocolFee)

		amountOut.Sub(amountOut, &stepProtocolFee)

		var stepLpFee utils.Uint256
		stepLpFee.Mul(amountOut, d.minFee).
			Div(&stepLpFee, SIX_DECIMALS_UI)
		lpFeeAmount.Add(lpFeeAmount, &stepLpFee)

		amountOut.Sub(amountOut, &stepLpFee)

		if sqrtPriceNextX96.Cmp(sqrtRatioTargetX96) == 0 || amountRemaining.Sign() == 0 {
			return nil
		}

		priceNextX96.Set(d.minFeeKinkPriceX96)
	}

	if lo.Ternary(
		zeroToOne,
		priceNextX96.Cmp(d.maxFeeKinkPriceX96) > 0,
		priceNextX96.Cmp(d.maxFeeKinkPriceX96) < 0,
	) {
		var stepAmountIn, stepAmountOut utils.Uint256

		err := computeSwapStepForSwapInWithoutFee(
			sqrtPriceNextX96,
			calculateStepTargetSqrtPriceX96(zeroToOne, d.maxFeeKinkPriceX96, sqrtRatioTargetX96),
			liquidity,
			amountRemaining,
			sqrtPriceNextX96,
			&stepAmountIn,
			&stepAmountOut,
			feeAmount,
		)
		if err != nil {
			return err
		}

		var stepAmountInSigned utils.Int256
		err = utils.ToInt256(amountIn, &stepAmountInSigned)
		if err != nil {
			return err
		}

		amountRemaining.Sub(amountRemaining, &stepAmountInSigned)

		var stepDynamicFee, priceEndX96 utils.Uint256
		err = utils.MulDivV2(sqrtPriceNextX96, sqrtPriceNextX96, Q96I, &priceEndX96, nil)
		if err != nil {
			return err
		}

		calculateStepDynamicFee(
			zeroToOne,
			&priceNextX96,
			&priceEndX96,
			d.zeroPriceImpactPriceX96,
			d.priceImpactToFeeDivisionFactor,
			&stepDynamicFee,
		)

		if stepAmountOut.Cmp(X86UI) > 0 {
			return ErrGreaterThanMaxAmountOut
		}

		var stepProtocolFee utils.Uint256
		stepProtocolFee.Mul(&stepAmountOut, protocolFee).
			Div(&stepProtocolFee, SIX_DECIMALS_UI)
		stepAmountOut.Sub(&stepAmountOut, &stepProtocolFee)

		var stepLpFee utils.Uint256
		stepLpFee.Mul(&stepAmountOut, &stepDynamicFee).
			Div(&stepLpFee, SIX_DECIMALS_UI)
		stepAmountOut.Sub(&stepAmountOut, &stepLpFee)

		amountOut.Add(amountOut, &stepAmountOut)
		amountIn.Add(amountIn, &stepAmountIn)
		protocolFeeAmount.Add(protocolFeeAmount, &stepProtocolFee)
		lpFeeAmount.Add(lpFeeAmount, &stepLpFee)

		if sqrtPriceNextX96.Cmp(sqrtRatioTargetX96) == 0 || amountRemaining.Sign() == 0 {
			return nil
		}
	}

	var stepAmountIn, stepAmountOut utils.Uint256

	err = computeSwapStepForSwapInWithoutFee(
		sqrtPriceNextX96,
		sqrtRatioTargetX96,
		liquidity,
		amountRemaining,
		sqrtPriceNextX96,
		&stepAmountIn,
		&stepAmountOut,
		feeAmount,
	)
	if err != nil {
		return err
	}

	if stepAmountOut.Cmp(X86UI) > 0 {
		return ErrGreaterThanMaxAmountOut
	}

	var stepProtocolFee utils.Uint256
	stepProtocolFee.Mul(&stepAmountOut, protocolFee).
		Div(&stepProtocolFee, SIX_DECIMALS_UI)
	stepAmountOut.Sub(&stepAmountOut, &stepProtocolFee)

	var stepLpFee utils.Uint256
	stepLpFee.Mul(&stepAmountOut, d.maxFee).
		Div(&stepLpFee, SIX_DECIMALS_UI)
	stepAmountOut.Sub(&stepAmountOut, &stepLpFee)

	amountOut.Add(amountOut, &stepAmountOut)
	amountIn.Add(amountIn, &stepAmountIn)
	protocolFeeAmount.Add(protocolFeeAmount, &stepProtocolFee)
	lpFeeAmount.Add(lpFeeAmount, &stepLpFee)

	return nil
}

func calculateStepTargetSqrtPriceX96(
	zeroToOne bool,
	sqrtPriceKinkX96 *utils.Uint256,
	sqrtPriceTargetX96 *utils.Uint256,
) *utils.Uint256 {
	if zeroToOne {
		if sqrtPriceKinkX96.Cmp(sqrtPriceTargetX96) < 0 {
			return sqrtPriceTargetX96
		}
		return sqrtPriceKinkX96
	}

	if sqrtPriceKinkX96.Cmp(sqrtPriceTargetX96) > 0 {
		return sqrtPriceTargetX96
	}
	return sqrtPriceKinkX96
}

func calculateStepDynamicFee(
	zeroToOne bool,
	priceStartX96 *utils.Uint256,
	priceEndX96 *utils.Uint256,
	zeroPriceImpactX96 *utils.Uint256,
	priceImpactToFeeDivisionFactor *utils.Uint256,

	stepDynamicFee *utils.Uint256,
) {
	var stepMeanPriceImpact, priceMeanX96 utils.Uint256

	priceMeanX96.Add(priceStartX96, priceEndX96).Rsh(&priceMeanX96, 1)

	if zeroToOne {
		stepMeanPriceImpact.Sub(zeroPriceImpactX96, &priceMeanX96)
	} else {
		stepMeanPriceImpact.Sub(&priceMeanX96, zeroPriceImpactX96)
	}
	stepMeanPriceImpact.Mul(&stepMeanPriceImpact, SIX_DECIMALS_UI).
		Div(&stepMeanPriceImpact, zeroPriceImpactX96)

	stepDynamicFee.Div(&stepMeanPriceImpact, priceImpactToFeeDivisionFactor)
}
