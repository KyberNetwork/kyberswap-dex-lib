package math

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	ErrMaxInRatio  = errors.New("MAX_IN_RATIO")
	ErrMaxOutRatio = errors.New("MAX_OUT_RATIO")

	_MAX_IN_RATIO  = uint256.NewInt(0.3e18)
	_MAX_OUT_RATIO = uint256.NewInt(0.3e18)
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
	maxIn, err := FixedPoint.MulDown(balanceIn, _MAX_IN_RATIO)
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
	maxIn, err := FixedPoint.MulDown(balanceIn, _MAX_IN_RATIO)
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
	maxOut, err := FixedPoint.MulDown(balanceOut, _MAX_OUT_RATIO)
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
	maxOut, err := FixedPoint.MulDown(balanceOut, _MAX_OUT_RATIO)
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
