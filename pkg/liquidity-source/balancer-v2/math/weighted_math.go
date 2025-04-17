package math

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	ErrMaxInRatio          = errors.New("MAX_IN_RATIO")
	ErrMaxOutRatio         = errors.New("MAX_OUT_RATIO")
	ErrZeroInvariant       = errors.New("ZERO_INVARIANT")
	ErrMinBptInForTokenOut = errors.New("MIN_BPT_IN_FOR_TOKEN_OUT")

	MAX_IN_RATIO                = uint256.NewInt(0.3e18)
	MAX_OUT_RATIO               = uint256.NewInt(0.3e18)
	_MIN_POW_BASE_FREE_EXPONENT = uint256.NewInt(0.7e18)
	_MIN_INVARIANT_RATIO        = uint256.NewInt(0.7e18)
)

var WeightedMath *weightedMath

type weightedMath struct {
}

func init() {
	WeightedMath = &weightedMath{}
}

// https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F9#L78
func (l *weightedMath) CalcOutGivenIn(
	balanceIn *uint256.Int,
	weightIn *uint256.Int,
	balanceOut *uint256.Int,
	weightOut *uint256.Int,
	amountIn *uint256.Int,
) (*uint256.Int, error) {
	maxIn, err := FixedPoint.MulDown(balanceIn, MAX_IN_RATIO)
	if err != nil {
		return nil, err
	}

	if amountIn.Gt(maxIn) {
		return nil, ErrMaxInRatio
	}

	denominator, err := FixedPoint.Add(balanceIn, amountIn)
	if err != nil {
		return nil, err
	}

	base, err := FixedPoint.DivUp(balanceIn, denominator)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivDown(weightIn, weightOut)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulDown(balanceOut, FixedPoint.Complement(power))
}

// https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F25#L69
func (l *weightedMath) CalcOutGivenInV1(
	balanceIn *uint256.Int,
	weightIn *uint256.Int,
	balanceOut *uint256.Int,
	weightOut *uint256.Int,
	amountIn *uint256.Int,
) (*uint256.Int, error) {
	maxIn, err := FixedPoint.MulDown(balanceIn, MAX_IN_RATIO)
	if err != nil {
		return nil, err
	}

	if amountIn.Gt(maxIn) {
		return nil, ErrMaxInRatio
	}

	denominator, err := FixedPoint.Add(balanceIn, amountIn)
	if err != nil {
		return nil, err
	}

	base, err := FixedPoint.DivUp(balanceIn, denominator)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivDown(weightIn, weightOut)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUpV1(base, exponent)
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulDown(balanceOut, FixedPoint.Complement(power))
}

// https://etherscan.io/address/0x065f5b35d4077334379847fe26f58b1029e51161#code#F9#L113
func (l *weightedMath) CalcInGivenOut(
	balanceIn *uint256.Int,
	weightIn *uint256.Int,
	balanceOut *uint256.Int,
	weightOut *uint256.Int,
	amountOut *uint256.Int,
) (*uint256.Int, error) {
	maxOut, err := FixedPoint.MulDown(balanceOut, MAX_OUT_RATIO)
	if err != nil {
		return nil, err
	}

	// Cannot exceed maximum out ratio
	if amountOut.Gt(maxOut) {
		return nil, ErrMaxOutRatio
	}

	remainingBalanceOut, err := FixedPoint.Sub(balanceOut, amountOut)
	if err != nil {
		return nil, err
	}

	base, err := FixedPoint.DivUp(balanceOut, remainingBalanceOut)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivUp(weightOut, weightIn)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	// Because the base is larger than one (and the power rounds up), the power should always be larger than one, so
	// the following subtraction should never revert.
	ratio, err := FixedPoint.Sub(power, FixedPoint.ONE)
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulUp(balanceIn, ratio)
}

// https://etherscan.io/address/0x6df50e37a6aefb9024a7284ef1c9e1e8e7c4f7b8#code#F25#L104
func (l *weightedMath) CalcInGivenOutV1(
	balanceIn *uint256.Int,
	weightIn *uint256.Int,
	balanceOut *uint256.Int,
	weightOut *uint256.Int,
	amountOut *uint256.Int,
) (*uint256.Int, error) {
	maxOut, err := FixedPoint.MulDown(balanceOut, MAX_OUT_RATIO)
	if err != nil {
		return nil, err
	}

	// Cannot exceed maximum out ratio
	if amountOut.Gt(maxOut) {
		return nil, ErrMaxOutRatio
	}

	remainingBalanceOut, err := FixedPoint.Sub(balanceOut, amountOut)
	if err != nil {
		return nil, err
	}

	base, err := FixedPoint.DivUp(balanceOut, remainingBalanceOut)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivUp(weightOut, weightIn)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUpV1(base, exponent)
	if err != nil {
		return nil, err
	}

	// Because the base is larger than one (and the power rounds up), the power should always be larger than one, so
	// the following subtraction should never revert.
	ratio, err := FixedPoint.Sub(power, FixedPoint.ONE)
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulUp(balanceIn, ratio)
}

func (l *weightedMath) CalculateInvariantV1(normalizedWeights, balances []*uint256.Int) (*uint256.Int, error) {
	invariant := new(uint256.Int).Set(FixedPoint.ONE)

	for i := range normalizedWeights {
		multiplier, err := FixedPoint.PowDownV1(balances[i], normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariant, err = FixedPoint.MulDown(invariant, multiplier)
		if err != nil {
			return nil, err
		}
	}

	if invariant.Sign() <= 0 {
		return nil, ErrZeroInvariant
	}

	return invariant, nil
}

func (l *weightedMath) CalculateInvariant(normalizedWeights, balances []*uint256.Int) (*uint256.Int, error) {
	invariant := new(uint256.Int).Set(FixedPoint.ONE)

	for i := range normalizedWeights {
		multiplier, err := FixedPoint.PowDown(balances[i], normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariant, err = FixedPoint.MulDown(invariant, multiplier)
		if err != nil {
			return nil, err
		}
	}

	if invariant.Sign() <= 0 {
		return nil, ErrZeroInvariant
	}

	return invariant, nil
}

func (l *weightedMath) CalcDueTokenProtocolSwapFeeAmountV1(
	balance *uint256.Int,
	normalizedWeight *uint256.Int,
	previousInvariant *uint256.Int,
	currentInvariant *uint256.Int,
	protocolSwapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	if currentInvariant.Cmp(previousInvariant) <= 0 {
		return uint256.NewInt(0), nil
	}

	base, err := FixedPoint.DivUp(previousInvariant, currentInvariant)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivDown(FixedPoint.ONE, normalizedWeight)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUpV1(Math.Max(base, _MIN_POW_BASE_FREE_EXPONENT), exponent)
	if err != nil {
		return nil, err
	}

	tokenAccruedFees, err := FixedPoint.MulDown(balance, FixedPoint.Complement(power))
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulDown(tokenAccruedFees, protocolSwapFeePercentage)
}

func (l *weightedMath) CalcDueTokenProtocolSwapFeeAmount(
	balance *uint256.Int,
	normalizedWeight *uint256.Int,
	previousInvariant *uint256.Int,
	currentInvariant *uint256.Int,
	protocolSwapFeePercentage *uint256.Int,
) (*uint256.Int, error) {
	if currentInvariant.Cmp(previousInvariant) <= 0 {
		return uint256.NewInt(0), nil
	}

	base, err := FixedPoint.DivUp(previousInvariant, currentInvariant)
	if err != nil {
		return nil, err
	}

	exponent, err := FixedPoint.DivDown(FixedPoint.ONE, normalizedWeight)
	if err != nil {
		return nil, err
	}

	power, err := FixedPoint.PowUp(Math.Max(base, _MIN_POW_BASE_FREE_EXPONENT), exponent)
	if err != nil {
		return nil, err
	}

	tokenAccruedFees, err := FixedPoint.MulDown(balance, FixedPoint.Complement(power))
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulDown(tokenAccruedFees, protocolSwapFeePercentage)
}

func (l *weightedMath) CalcBptOutGivenExactTokensIn(
	balances []*uint256.Int,
	normalizedWeights []*uint256.Int,
	amountsIn []*uint256.Int,
	bptTotalSupply *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	balanceRatiosWithFee := make([]*uint256.Int, len(balances))

	invariantRatioWithFees := uint256.NewInt(0)
	tmp := new(uint256.Int)

	var err error
	for i := range balances {
		balanceRatiosWithFee[i], err = FixedPoint.Add(balances[i], amountsIn[i])
		if err != nil {
			return nil, err
		}

		balanceRatiosWithFee[i], err = FixedPoint.DivDown(balanceRatiosWithFee[i], balances[i])
		if err != nil {
			return nil, err
		}

		tmp, err = FixedPoint.MulDown(balanceRatiosWithFee[i], normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariantRatioWithFees, err = FixedPoint.Add(invariantRatioWithFees, tmp)
		if err != nil {
			return nil, err
		}
	}

	invariantRatio := new(uint256.Int).Set(FixedPoint.ONE)
	nonTaxableAmount := new(uint256.Int)
	taxableAmount := new(uint256.Int)
	amountInWithoutFee := new(uint256.Int)
	balanceRatio := new(uint256.Int)

	for i := range balances {
		amountInWithoutFee.SetUint64(0)

		if balanceRatiosWithFee[i].Gt(invariantRatioWithFees) {
			tmp, err := FixedPoint.Sub(invariantRatioWithFees, FixedPoint.ONE)
			if err != nil {
				return nil, err
			}

			nonTaxableAmount, err = FixedPoint.MulDown(balances[i], tmp)
			if err != nil {
				return nil, err
			}

			taxableAmount.Sub(amountsIn[i], nonTaxableAmount)
			if taxableAmount.Sign() < 0 {
				taxableAmount.SetUint64(0)
			}

			tmp, err = FixedPoint.Sub(FixedPoint.ONE, swapFee)
			if err != nil {
				return nil, err
			}

			tmp, err = FixedPoint.MulDown(taxableAmount, tmp)
			if err != nil {
				return nil, err
			}

			amountInWithoutFee.Add(nonTaxableAmount, tmp)
		} else {
			amountInWithoutFee.Set(amountsIn[i])
		}

		balanceRatio.Add(balances[i], amountInWithoutFee)
		balanceRatio, err := FixedPoint.DivDown(balanceRatio, balances[i])
		if err != nil {
			return nil, err
		}

		tmp, err = FixedPoint.PowDown(balanceRatio, normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariantRatio, err = FixedPoint.MulDown(invariantRatio, tmp)
		if err != nil {
			return nil, err
		}
	}

	if invariantRatio.Cmp(FixedPoint.ONE) >= 0 {
		tmp, err := FixedPoint.Sub(invariantRatio, FixedPoint.ONE)
		if err != nil {
			return nil, err
		}

		return FixedPoint.MulDown(bptTotalSupply, tmp)
	}

	return uint256.NewInt(0), nil
}

func (l *weightedMath) CalcBptOutGivenExactTokensInV1(
	balances []*uint256.Int,
	normalizedWeights []*uint256.Int,
	amountsIn []*uint256.Int,
	bptTotalSupply *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	balanceRatiosWithFee := make([]*uint256.Int, len(balances))

	invariantRatioWithFees := uint256.NewInt(0)
	tmp := new(uint256.Int)

	var err error
	for i := range balances {
		balanceRatiosWithFee[i], err = FixedPoint.Add(balances[i], amountsIn[i])
		if err != nil {
			return nil, err
		}

		balanceRatiosWithFee[i], err = FixedPoint.DivDown(balanceRatiosWithFee[i], balances[i])
		if err != nil {
			return nil, err
		}

		tmp, err = FixedPoint.MulDown(balanceRatiosWithFee[i], normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariantRatioWithFees, err = FixedPoint.Add(invariantRatioWithFees, tmp)
		if err != nil {
			return nil, err
		}
	}

	invariantRatio := new(uint256.Int).Set(FixedPoint.ONE)
	nonTaxableAmount := new(uint256.Int)
	taxableAmount := new(uint256.Int)
	amountInWithoutFee := new(uint256.Int)
	balanceRatio := new(uint256.Int)

	for i := range balances {
		amountInWithoutFee.SetUint64(0)

		if balanceRatiosWithFee[i].Gt(invariantRatioWithFees) {
			tmp, err := FixedPoint.Sub(invariantRatioWithFees, FixedPoint.ONE)
			if err != nil {
				return nil, err
			}

			nonTaxableAmount, err = FixedPoint.MulDown(balances[i], tmp)
			if err != nil {
				return nil, err
			}

			taxableAmount.Sub(amountsIn[i], nonTaxableAmount)
			if taxableAmount.Sign() < 0 {
				taxableAmount.SetUint64(0)
			}

			tmp, err = FixedPoint.Sub(FixedPoint.ONE, swapFee)
			if err != nil {
				return nil, err
			}

			tmp, err = FixedPoint.MulDown(taxableAmount, tmp)
			if err != nil {
				return nil, err
			}

			amountInWithoutFee.Add(nonTaxableAmount, tmp)
		} else {
			amountInWithoutFee.Set(amountsIn[i])
		}

		balanceRatio.Add(balances[i], amountInWithoutFee)
		balanceRatio, err := FixedPoint.DivDown(balanceRatio, balances[i])
		if err != nil {
			return nil, err
		}

		tmp, err = FixedPoint.PowDownV1(balanceRatio, normalizedWeights[i])
		if err != nil {
			return nil, err
		}

		invariantRatio, err = FixedPoint.MulDown(invariantRatio, tmp)
		if err != nil {
			return nil, err
		}
	}

	if invariantRatio.Cmp(FixedPoint.ONE) >= 0 {
		tmp, err := FixedPoint.Sub(invariantRatio, FixedPoint.ONE)
		if err != nil {
			return nil, err
		}

		return FixedPoint.MulDown(bptTotalSupply, tmp)
	}

	return uint256.NewInt(0), nil
}

func (w *weightedMath) CalcTokenOutGivenExactBptInV1(
	balance *uint256.Int,
	normalizedWeight *uint256.Int,
	bptAmountIn *uint256.Int,
	bptTotalSupply *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	/*****************************************************************************************
	// exactBPTInForTokenOut                                                                //
	// a = amountOut                                                                        //
	// b = balance                     /      /    totalBPT - bptIn       \    (1 / w)  \   //
	// bptIn = bptAmountIn    a = b * |  1 - | --------------------------  | ^           |  //
	// bpt = totalBPT                  \      \       totalBPT            /             /   //
	// w = weight                                                                           //
	*****************************************************************************************/

	tmp := new(uint256.Int)

	tmp.Sub(bptTotalSupply, bptAmountIn)
	invariantRatio, err := FixedPoint.DivUp(tmp, bptTotalSupply)
	if err != nil {
		return nil, err
	}

	if invariantRatio.Cmp(_MIN_INVARIANT_RATIO) < 0 {
		return nil, ErrMinBptInForTokenOut
	}

	tmp, err = FixedPoint.DivDown(FixedPoint.ONE, normalizedWeight)
	if err != nil {
		return nil, err
	}

	balanceRatio, err := FixedPoint.PowUpV1(invariantRatio, tmp)
	if err != nil {
		return nil, err
	}

	amountOutWithoutFee, err := FixedPoint.MulDown(balance, FixedPoint.Complement(balanceRatio))
	if err != nil {
		return nil, err
	}

	taxableAmount, err := FixedPoint.MulUp(amountOutWithoutFee, FixedPoint.Complement(normalizedWeight))
	if err != nil {
		return nil, err
	}

	nonTaxableAmount := new(uint256.Int).Sub(amountOutWithoutFee, taxableAmount)

	tmp, err = FixedPoint.MulDown(taxableAmount, FixedPoint.Complement(swapFee))
	if err != nil {
		return nil, err
	}

	return new(uint256.Int).Add(nonTaxableAmount, tmp), nil
}

func (w *weightedMath) CalcTokenOutGivenExactBptIn(
	balance *uint256.Int,
	normalizedWeight *uint256.Int,
	bptAmountIn *uint256.Int,
	bptTotalSupply *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	/*****************************************************************************************
	// exactBPTInForTokenOut                                                                //
	// a = amountOut                                                                        //
	// b = balance                     /      /    totalBPT - bptIn       \    (1 / w)  \   //
	// bptIn = bptAmountIn    a = b * |  1 - | --------------------------  | ^           |  //
	// bpt = totalBPT                  \      \       totalBPT            /             /   //
	// w = weight                                                                           //
	*****************************************************************************************/

	tmp := new(uint256.Int)

	tmp.Sub(bptTotalSupply, bptAmountIn)
	invariantRatio, err := FixedPoint.DivUp(tmp, bptTotalSupply)
	if err != nil {
		return nil, err
	}

	if invariantRatio.Cmp(_MIN_INVARIANT_RATIO) < 0 {
		return nil, ErrMinBptInForTokenOut
	}

	tmp, err = FixedPoint.DivDown(FixedPoint.ONE, normalizedWeight)
	if err != nil {
		return nil, err
	}

	balanceRatio, err := FixedPoint.PowUp(invariantRatio, tmp)
	if err != nil {
		return nil, err
	}

	amountOutWithoutFee, err := FixedPoint.MulDown(balance, FixedPoint.Complement(balanceRatio))
	if err != nil {
		return nil, err
	}

	taxableAmount, err := FixedPoint.MulUp(amountOutWithoutFee, FixedPoint.Complement(normalizedWeight))
	if err != nil {
		return nil, err
	}

	nonTaxableAmount := new(uint256.Int).Sub(amountOutWithoutFee, taxableAmount)

	tmp, err = FixedPoint.MulDown(taxableAmount, FixedPoint.Complement(swapFee))
	if err != nil {
		return nil, err
	}

	return new(uint256.Int).Add(nonTaxableAmount, tmp), nil
}
