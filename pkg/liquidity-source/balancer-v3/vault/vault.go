package vault

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
)

type Vault struct {
	balancesLiveScaled18       []*uint256.Int
	decimalScalingFactors      []*uint256.Int
	tokenRates                 []*uint256.Int
	staticSwapFeePercentage    *uint256.Int
	aggregateSwapFeePercentage *uint256.Int

	hook        hooks.IHook
	hooksConfig shared.HooksConfig

	isPoolInRecoveryMode bool
}

func New(hook hooks.IHook, hooksConfig shared.HooksConfig,
	decimalScalingFactors, tokenRates, balancesLiveScaled18 []*uint256.Int,
	swapFeePercentage, aggregateSwapFeePercentage *uint256.Int) *Vault {
	return &Vault{
		hook:                       hook,
		hooksConfig:                hooksConfig,
		decimalScalingFactors:      decimalScalingFactors,
		tokenRates:                 tokenRates,
		balancesLiveScaled18:       balancesLiveScaled18,
		staticSwapFeePercentage:    swapFeePercentage,
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

// Swap per https://etherscan.io/address/0xbA1333333333a1BA1108E8412f11850A5C319bA9#code#F1#L197
func (v *Vault) Swap(vaultSwapParams shared.VaultSwapParams, onSwap shared.OnSwapFn) (*uint256.Int, *uint256.Int,
	*uint256.Int, error) {
	amountGivenScaled18, err := v.ComputeAmountGivenScaled18(vaultSwapParams)
	if err != nil {
		return nil, nil, nil, err
	}

	var poolSwapParams = shared.PoolSwapParams{
		Kind:                    vaultSwapParams.Kind,
		OnSwap:                  onSwap,
		StaticSwapFeePercentage: v.staticSwapFeePercentage,
		AmountGivenScaled18:     amountGivenScaled18,
		BalancesScaled18:        v.balancesLiveScaled18,
		IndexIn:                 vaultSwapParams.IndexIn,
		IndexOut:                vaultSwapParams.IndexOut,
	}

	if v.hooksConfig.ShouldCallBeforeSwap {
		if err := v.callBeforeSwapHook(poolSwapParams); err != nil {
			return nil, nil, nil, err
		}

		// WARN: some states can be changed after hook
	}

	if v.hooksConfig.ShouldCallComputeDynamicSwapFee {
		poolSwapParams.StaticSwapFeePercentage, err = v.callComputeDynamicSwapFeeHook(poolSwapParams)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var totalSwapFeeAmountScaled18 *uint256.Int
	if vaultSwapParams.Kind == shared.ExactIn {
		totalSwapFeeAmountScaled18, err = math.FixPoint.MulUp(poolSwapParams.AmountGivenScaled18,
			poolSwapParams.StaticSwapFeePercentage)
		if err != nil {
			return nil, nil, nil, err
		}

		poolSwapParams.AmountGivenScaled18, err = math.FixPoint.Sub(poolSwapParams.AmountGivenScaled18,
			totalSwapFeeAmountScaled18)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// _ensureValidSwapAmount
	if amountGivenScaled18.Lt(MinimumTradeAmount) {
		return nil, nil, nil, ErrAmountInTooSmall
	}

	amountCalculatedScaled18, err := onSwap(poolSwapParams)
	if err != nil {
		return nil, nil, nil, err
	}

	// _ensureValidSwapAmount
	if amountCalculatedScaled18.Lt(MinimumTradeAmount) {
		return nil, nil, nil, ErrAmountOutTooSmall
	}

	var amountCalculated *uint256.Int
	if vaultSwapParams.Kind == shared.ExactIn {
		if amountCalculated, err = toRawUndoRateRoundDown(
			amountCalculatedScaled18,
			v.decimalScalingFactors[poolSwapParams.IndexOut],
			computeRateRoundUp(v.tokenRates[poolSwapParams.IndexOut]),
		); err != nil {
			return nil, nil, nil, err
		}
	} else {
		totalSwapFeeAmountScaled18, err = math.FixPoint.MulDivUp(amountCalculatedScaled18, v.staticSwapFeePercentage,
			math.FixPoint.Complement(v.staticSwapFeePercentage))
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

	totalSwapFee, aggregateFee, err := v.ComputeAggregateSwapFees(poolSwapParams.IndexIn, totalSwapFeeAmountScaled18,
		v.aggregateSwapFeePercentage)
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

func (v *Vault) ComputeAmountGivenScaled18(param shared.VaultSwapParams) (*uint256.Int, error) {
	if param.Kind == shared.ExactIn {
		return toScaled18ApplyRateRoundDown(param.AmountGivenRaw, v.decimalScalingFactors[param.IndexIn],
			v.tokenRates[param.IndexIn])
	}

	return toScaled18ApplyRateRoundUp(param.AmountGivenRaw, v.decimalScalingFactors[param.IndexOut],
		computeRateRoundUp(v.tokenRates[param.IndexOut]))
}

func (v *Vault) ComputeAggregateSwapFees(index int, totalSwapFeeAmountScaled18, aggregateSwapFeePercentage *uint256.Int,
) (totalSwapFeeAmountRaw, aggregateSwapFeeAmountRaw *uint256.Int, err error) {
	if totalSwapFeeAmountScaled18.Sign() > 0 {
		totalSwapFeeAmountRaw, err = toRawUndoRateRoundDown(totalSwapFeeAmountScaled18, v.decimalScalingFactors[index],
			v.tokenRates[index])
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

		return totalSwapFeeAmountRaw, math.U0, nil
	}

	return math.U0, math.U0, nil
}

func (v *Vault) UpdateLiveBalance(
	index int,
	amountGivenRaw *uint256.Int,
	rounding shared.Rounding,
) (newBalanceLiveScaled18 *uint256.Int, err error) {
	if rounding == shared.RoundUp {
		newBalanceLiveScaled18, err = toScaled18ApplyRateRoundUp(amountGivenRaw, v.decimalScalingFactors[index],
			v.tokenRates[index])
	} else {
		newBalanceLiveScaled18, err = toScaled18ApplyRateRoundDown(amountGivenRaw, v.decimalScalingFactors[index],
			v.tokenRates[index])
	}

	if err != nil {
		return nil, err
	}

	v.balancesLiveScaled18[index] = newBalanceLiveScaled18

	return
}
