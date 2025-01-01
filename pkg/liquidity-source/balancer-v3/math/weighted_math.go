package math

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	ErrMaxInRatio  = errors.New("MAX_IN_RATIO")
	ErrMaxOutRatio = errors.New("MAX_OUT_RATIO")

	MAX_IN_RATIO  = uint256.NewInt(30e16)
	MAX_OUT_RATIO = uint256.NewInt(30e16)
)

var WeightedMath *weightedMath

type weightedMath struct{}

func init() {
	WeightedMath = &weightedMath{}
}

func (s *weightedMath) ComputeOutGivenExactIn(
	balanceIn,
	weightIn,
	balanceOut,
	weightOut,
	amountIn *uint256.Int,
) (*uint256.Int, error) {
	/**********************************************************************************************
	  // inGivenExactOut                                                                           //
	  // aO = amountOut                                                                            //
	  // bO = balanceOut                                                                           //
	  // bI = balanceIn              /  /            bO             \    (wO / wI)      \          //
	  // aI = amountIn    aI = bI * |  | --------------------------  | ^            - 1  |         //
	  // wI = weightIn               \  \       ( bO - aO )         /                   /          //
	  // wO = weightOut                                                                            //
	  **********************************************************************************************/

	balanceInApplyRate, err := FixPoint.MulDown(amountIn, MAX_IN_RATIO)
	if err != nil {
		return nil, err
	}

	if amountIn.Gt(balanceInApplyRate) {
		return nil, ErrMaxInRatio
	}

	denominator, err := FixPoint.Add(balanceIn, amountIn)
	if err != nil {
		return nil, err
	}

	base, err := FixPoint.DivUp(balanceIn, denominator)
	if err != nil {
		return nil, err
	}

	exponent, err := FixPoint.DivDown(weightIn, weightOut)
	if err != nil {
		return nil, err
	}

	power, err := FixPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	return FixPoint.MulDown(balanceOut, FixPoint.Complement(power))
}

func (s *weightedMath) ComputeInGivenExactOut(
	balanceIn,
	weightIn,
	balanceOut,
	weightOut,
	amountOut *uint256.Int,
) (*uint256.Int, error) {
	/**********************************************************************************************
	  // outGivenExactIn                                                                           //
	  // aO = amountOut                                                                            //
	  // bO = balanceOut                                                                           //
	  // bI = balanceIn              /      /            bI             \    (wI / wO) \           //
	  // aI = amountIn    aO = bO * |  1 - | --------------------------  | ^            |          //
	  // wI = weightIn               \      \       ( bI + aI )         /              /           //
	  // wO = weightOut                                                                            //
	  **********************************************************************************************/

	balanceOutApplyRate, err := FixPoint.MulDown(amountOut, MAX_OUT_RATIO)
	if err != nil {
		return nil, err
	}

	if amountOut.Gt(balanceOutApplyRate) {
		return nil, ErrMaxOutRatio
	}

	delta, err := FixPoint.Sub(balanceOut, amountOut)
	if err != nil {
		return nil, err
	}

	base, err := FixPoint.DivUp(balanceOut, delta)
	if err != nil {
		return nil, err
	}

	exponent, err := FixPoint.DivUp(weightOut, weightIn)
	if err != nil {
		return nil, err
	}

	power, err := FixPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	ratio, err := FixPoint.Sub(power, ONE_E18)
	if err != nil {
		return nil, err
	}

	return FixPoint.MulUp(balanceIn, ratio)
}
