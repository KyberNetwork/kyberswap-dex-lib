package vault

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type Vault struct {
	balancesLiveScaled18       []*uint256.Int
	decimalScalingFactors      []*uint256.Int
	tokenRates                 []*uint256.Int
	swapFeePercentage          *uint256.Int
	aggregateSwapFeePercentage *uint256.Int

	hook        hooks.IHook
	hooksConfig shared.HooksConfig

	isPoolInRecoveryMode bool
}

func New(hook hooks.IHook, hooksConfig shared.HooksConfig, isPoolInRecoveryMode bool,
	decimalScalingFactors, tokenRates, balancesLiveScaled18 []*uint256.Int,
	swapFeePercentage, aggregateSwapFeePercentage *uint256.Int,
) *Vault {
	return &Vault{
		hook:                       hook,
		hooksConfig:                hooksConfig,
		decimalScalingFactors:      decimalScalingFactors,
		tokenRates:                 tokenRates,
		balancesLiveScaled18:       balancesLiveScaled18,
		swapFeePercentage:          swapFeePercentage,
		aggregateSwapFeePercentage: aggregateSwapFeePercentage,
	}
}

func (v *Vault) CloneState() *Vault {
	cloned := *v

	v.balancesLiveScaled18 = lo.Map(v.balancesLiveScaled18, func(v *uint256.Int, _ int) *uint256.Int {
		return new(uint256.Int).Set(v)
	})

	return &cloned
}

func (v *Vault) Swap(
	vaultSwapParams shared.VaultSwapParams,
	onSwap func(param shared.PoolSwapParams) (*uint256.Int, error),
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	amountGivenScaled18, err := v.ComputeAmountGivenScaled18(vaultSwapParams.Kind, vaultSwapParams.AmountGivenRaw,
		v.decimalScalingFactors[vaultSwapParams.IndexOut], v.tokenRates[vaultSwapParams.IndexOut])
	if err != nil {
		return nil, nil, nil, err
	}

	var poolSwapParams = shared.PoolSwapParams{
		Kind:                 vaultSwapParams.Kind,
		SwapFeePercentage:    v.swapFeePercentage,
		AmountGivenScaled18:  amountGivenScaled18,
		BalancesLiveScaled18: v.balancesLiveScaled18,
		IndexIn:              vaultSwapParams.IndexIn,
		IndexOut:             vaultSwapParams.IndexOut,
	}

	if v.hooksConfig.ShouldCallBeforeSwap {
		if err := v.callBeforeSwapHook(poolSwapParams); err != nil {
			return nil, nil, nil, err
		}

		// WARN: some states can be changed after hook
	}

	if v.hooksConfig.ShouldCallComputeDynamicSwapFee {
		swapFeePercentage, err := v.callComputeDynamicSwapFeeHook(poolSwapParams)
		if err != nil {
			return nil, nil, nil, err
		}

		poolSwapParams.SwapFeePercentage = swapFeePercentage
	}

	var totalSwapFeeAmountScaled18 *uint256.Int
	if vaultSwapParams.Kind == shared.EXACT_IN {
		totalSwapFeeAmountScaled18, err = math.FixPoint.MulUp(poolSwapParams.AmountGivenScaled18, poolSwapParams.SwapFeePercentage)
		if err != nil {
			return nil, nil, nil, err
		}

		poolSwapParams.AmountGivenScaled18, err = math.FixPoint.Sub(poolSwapParams.AmountGivenScaled18, totalSwapFeeAmountScaled18)
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

	var amountCalculated *uint256.Int
	if vaultSwapParams.Kind == shared.EXACT_IN {
		if amountCalculated, err = toRawUndoRateRoundDown(
			amountCalculatedScaled18,
			v.decimalScalingFactors[poolSwapParams.IndexOut],
			computeRateRoundUp(v.tokenRates[poolSwapParams.IndexOut]),
		); err != nil {
			return nil, nil, nil, err
		}
	} else {
		totalSwapFeeAmountScaled18, err := math.FixPoint.MulDivUp(amountCalculatedScaled18, v.swapFeePercentage, math.FixPoint.Complement(v.swapFeePercentage))
		if err != nil {
			return nil, nil, nil, err
		}

		amountCalculatedScaled18, err = math.FixPoint.Add(amountCalculatedScaled18, totalSwapFeeAmountScaled18)
		if err != nil {
			return nil, nil, nil, err
		}

		if amountCalculated, err = toRawUndoRateRoundDown(
			amountCalculatedScaled18,
			v.decimalScalingFactors[poolSwapParams.IndexIn],
			v.tokenRates[poolSwapParams.IndexIn],
		); err != nil {
			return nil, nil, nil, err
		}
	}

	totalSwapFee, aggregateFee, err := v.ComputeAggregateSwapFees(totalSwapFeeAmountScaled18, v.aggregateSwapFeePercentage,
		v.decimalScalingFactors[poolSwapParams.IndexIn], v.tokenRates[poolSwapParams.IndexIn])
	if err != nil {
		return nil, nil, nil, err
	}

	if v.hooksConfig.ShouldCallAfterSwap {
		amountCalculated, err = v.callAfterSwapHook(vaultSwapParams, poolSwapParams.AmountGivenScaled18,
			amountCalculatedScaled18, amountCalculated)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return amountCalculated, totalSwapFee, aggregateFee, nil
}

func (v *Vault) ComputeAmountGivenScaled18(kind shared.SwapKind, amountGiven, decimalScalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	if kind == shared.EXACT_IN {
		return toScaled18ApplyRateRoundDown(amountGiven, decimalScalingFactor, tokenRate)
	}

	return toScaled18ApplyRateRoundUp(amountGiven, decimalScalingFactor, computeRateRoundUp(tokenRate))
}

func (v *Vault) ComputeAggregateSwapFees(
	totalSwapFeeAmountScaled18, aggregateSwapFeePercentage,
	decimalScalingFactor, tokenRate *uint256.Int,
) (totalSwapFeeAmountRaw, aggregateSwapFeeAmountRaw *uint256.Int, err error) {
	if totalSwapFeeAmountScaled18.Sign() > 0 {
		totalSwapFeeAmountRaw, err = toRawUndoRateRoundDown(totalSwapFeeAmountScaled18, decimalScalingFactor, tokenRate)
		if err != nil {
			return nil, nil, err
		}

		if !v.isPoolInRecoveryMode {
			aggregateSwapFeeAmountRaw, err = math.FixPoint.MulDown(totalSwapFeeAmountRaw, aggregateSwapFeePercentage)
			if err != nil {
				return nil, nil, err
			}

			if aggregateSwapFeeAmountRaw.Gt(totalSwapFeeAmountRaw) {
				return nil, nil, ErrProtocolFeesExceedTotalCollected
			}

			return
		}

		return totalSwapFeeAmountRaw, math.ZERO, nil
	}

	return math.ZERO, math.ZERO, nil
}

func (v *Vault) UpdateLiveBalance(
	index int,
	amountGivenRaw *uint256.Int,
	rounding shared.Rounding,
) (newBalanceLiveScaled18 *uint256.Int, err error) {
	if rounding == shared.ROUND_UP {
		newBalanceLiveScaled18, err = toScaled18ApplyRateRoundUp(amountGivenRaw, v.decimalScalingFactors[index], v.tokenRates[index])
	} else {
		newBalanceLiveScaled18, err = toScaled18ApplyRateRoundDown(amountGivenRaw, v.decimalScalingFactors[index], v.tokenRates[index])
	}

	if err != nil {
		return nil, err
	}

	v.balancesLiveScaled18[index] = newBalanceLiveScaled18

	return
}
