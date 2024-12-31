package vault

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
)

type Vault struct {
	hook                       hooks.IHook
	hooksConfig                shared.HooksConfig
	decimalScalingFactors      []*uint256.Int
	tokenRates                 []*uint256.Int
	balancesLiveScaled18       []*uint256.Int
	amplificationParameter     *uint256.Int
	swapFeePercentage          *uint256.Int
	aggregateSwapFeePercentage *uint256.Int
}

func NewVault(hook hooks.IHook, hooksConfig shared.HooksConfig,
	decimalScalingFactors, tokenRates, balancesLiveScaled18 []*uint256.Int,
	amplificationParameter, swapFeePercentage, aggregateSwapFeePercentage *uint256.Int,
) *Vault {
	return &Vault{
		hook:                       hook,
		hooksConfig:                hooksConfig,
		decimalScalingFactors:      decimalScalingFactors,
		tokenRates:                 tokenRates,
		balancesLiveScaled18:       balancesLiveScaled18,
		amplificationParameter:     amplificationParameter,
		swapFeePercentage:          swapFeePercentage,
		aggregateSwapFeePercentage: aggregateSwapFeePercentage,
	}
}

func (v *Vault) Swap(
	vaultSwapParams shared.VaultSwapParams,
	onSwap func(param shared.PoolSwapParams) (*uint256.Int, error),
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	amountGivenScaled18, err := v.ComputeAmountGivenScaled18(true, vaultSwapParams.AmountGivenRaw,
		v.decimalScalingFactors[vaultSwapParams.IndexOut], v.tokenRates[vaultSwapParams.IndexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	var poolSwapParams = shared.PoolSwapParams{
		Kind:                 vaultSwapParams.Kind,
		AmountGivenScaled18:  amountGivenScaled18,
		BalancesLiveScaled18: v.balancesLiveScaled18,
		IndexIn:              vaultSwapParams.IndexIn,
		IndexOut:             vaultSwapParams.IndexOut,
	}

	if v.hooksConfig.ShouldCallBeforeSwap {
		v.hook.OnBeforeSwap()

	}
	if v.hooksConfig.ShouldCallComputeDynamicSwapFee {
		swapFeePercentage, err := v.callComputeDynamicSwapFeeHook(poolSwapParams)
		if err != nil {
			return nil, nil, nil, err
		}

		poolSwapParams.SwapFeePercentage = swapFeePercentage
	}

	if vaultSwapParams.Kind == shared.EXACT_IN {
		totalSwapFeeAmountScaled18, err := math.MulUp(poolSwapParams.AmountGivenScaled18, poolSwapParams.SwapFeePercentage)
		if err != nil {
			return nil, nil, nil, err
		}

		poolSwapParams.AmountGivenScaled18, err = math.Sub(poolSwapParams.AmountGivenScaled18, totalSwapFeeAmountScaled18)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// _ensureValidSwapAmount
	if amountGivenScaled18.Lt(MINIMUM_TRADE_AMOUNT) {
		return nil, nil, nil, ErrTradeAmountTooSmall
	}

	amountCalculatedScaled18, err := onSwap(poolSwapParams)
	if err != nil {
		return nil, nil, nil, err
	}

	// _ensureValidSwapAmount
	if amountCalculatedScaled18.Lt(MINIMUM_TRADE_AMOUNT) {
		return nil, nil, nil, ErrTradeAmountTooSmall
	}

	if vaultSwapParams.Kind == shared.EXACT_IN {

	}

	amountCalculated, err := v.ComputeAmountCalculatedRaw(true, amountGivenScaled18, v.swapFeePercentage,
		v.decimalScalingFactors[param.IndexOut], v.tokenRates[param.IndexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	totalSwapFee, aggregateFee, err := v.ComputeAggregateSwapFees(true, swapFeeScaled18, v.aggregateSwapFeePercentage,
		param.DecimalScalingFactor, param.TokenRate)
	if err != nil {
		return nil, nil, nil, err
	}

	return amountCalculated, totalSwapFee, aggregateFee, nil
}

func (v *Vault) ComputeAmountGivenScaled18(isExactIn bool, amountGiven, decimalScalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	if isExactIn {
		return toScaled18ApplyRateRoundDown(amountGiven, decimalScalingFactor, tokenRate)
	}

	return toScaled18ApplyRateRoundUp(amountGiven, decimalScalingFactor, computeRateRoundUp(tokenRate))
}

func (v *Vault) ComputeAmountCalculatedRaw(
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

func (v *Vault) ComputeAggregateSwapFees(
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

func (v *Vault) UpdateLiveBalance(
	param shared.VaultSwapParams,
	rounding shared.Rounding,
) (*uint256.Int, error) {
	if rounding == shared.ROUND_UP {
		return toScaled18ApplyRateRoundUp(param.AmountGiven, param.DecimalScalingFactor, param.TokenRate)
	}

	return toScaled18ApplyRateRoundDown(param.AmountGiven, param.DecimalScalingFactor, param.TokenRate)
}
