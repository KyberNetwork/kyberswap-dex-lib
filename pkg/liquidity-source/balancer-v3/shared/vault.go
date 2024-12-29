package shared

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/holiman/uint256"
)

func Swap(
	param VaultSwapParams,
	onSwap func(_ bool, _, _ int, _ *uint256.Int) (*uint256.Int, error),
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	amountGivenScaled18, err := ComputeAmountGivenScaled18(true, param.AmountGiven, param.DecimalScalingFactor, param.TokenRate)
	if err != nil {
		return nil, nil, nil, err
	}

	swapFeeScaled18, err := math.MulUp(amountGivenScaled18, param.SwapFeePercentage)
	if err != nil {
		return nil, nil, nil, err
	}

	amountGivenScaled18, err = math.Sub(amountGivenScaled18, swapFeeScaled18)
	if err != nil {
		return nil, nil, nil, err
	}

	if amountGivenScaled18.Lt(MINIMUM_TRADE_AMOUNT) {
		return nil, nil, nil, ErrTradeAmountTooSmall
	}

	amountCalculatedScaled18, err := onSwap(param.IsExactIn, param.IndexIn, param.IndexOut, amountGivenScaled18)
	if err != nil {
		return nil, nil, nil, err
	}

	if amountCalculatedScaled18.Lt(MINIMUM_TRADE_AMOUNT) {
		return nil, nil, nil, ErrTradeAmountTooSmall
	}

	amountCalculated, err := ComputeAmountCalculatedRaw(true, amountGivenScaled18, param.SwapFeePercentage, param.DecimalScalingFactor, param.TokenRate)
	if err != nil {
		return nil, nil, nil, err
	}

	totalSwapFee, aggregateFee, err := ComputeAggregateSwapFees(true, swapFeeScaled18, param.AggregateSwapFeePercentage,
		param.DecimalScalingFactor, param.TokenRate)
	if err != nil {
		return nil, nil, nil, err
	}

	return amountCalculated, totalSwapFee, aggregateFee, nil
}

func ComputeAmountGivenScaled18(isExactIn bool, amountGiven, decimalScalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	if isExactIn {
		return toScaled18ApplyRateRoundDown(amountGiven, decimalScalingFactor, tokenRate)
	}

	return toScaled18ApplyRateRoundUp(amountGiven, decimalScalingFactor, computeRateRoundUp(tokenRate))
}

func ComputeAmountCalculatedRaw(
	isExactIn bool,
	amountCalculatedScaled18, swapFeePercentage,
	decimalScalingFactor, tokenRate *uint256.Int,
) (*uint256.Int, error) {
	if isExactIn {
		return toRawUndoRateRoundDown(amountCalculatedScaled18, decimalScalingFactor, computeRateRoundUp(tokenRate))
	}

	totalSwapFeeAmountScaled18, err := math.MulDivUp(amountCalculatedScaled18, swapFeePercentage, math.Complement(swapFeePercentage))
	if err != nil {
		return nil, err
	}

	amountCalculatedScaled18, err = math.Add(amountCalculatedScaled18, totalSwapFeeAmountScaled18)
	if err != nil {
		return nil, err
	}

	return toRawUndoRateRoundDown(amountCalculatedScaled18, decimalScalingFactor, tokenRate)
}

func ComputeAggregateSwapFees(
	isExactIn bool,
	totalSwapFeeAmountScaled18, aggregateSwapFeePercentage,
	decimalScalingFactor, tokenRate *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	if totalSwapFeeAmountScaled18.IsZero() {
		return math.ZERO, math.ZERO, nil
	}

	totalSwapFeeAmountRaw, err := toRawUndoRateRoundDown(totalSwapFeeAmountScaled18, decimalScalingFactor, tokenRate)
	if err != nil {
		return nil, nil, err
	}

	// should check if pool is in Recovery Mode
	aggregateFeeAmountRaw, err := math.MulDown(totalSwapFeeAmountRaw, aggregateSwapFeePercentage)
	if err != nil {
		return nil, nil, err
	}

	if aggregateFeeAmountRaw.Gt(totalSwapFeeAmountRaw) {
		return nil, nil, ErrProtocolFeesExceedTotalCollected
	}

	return totalSwapFeeAmountRaw, aggregateFeeAmountRaw, nil
}

func UpdateLiveBalance(
	param VaultSwapParams,
	rounding Rounding,
) (*uint256.Int, error) {
	if rounding == ROUND_UP {
		return toScaled18ApplyRateRoundUp(param.AmountGiven, param.DecimalScalingFactor, param.TokenRate)
	}

	return toScaled18ApplyRateRoundDown(param.AmountGiven, param.DecimalScalingFactor, param.TokenRate)
}
