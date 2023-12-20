package math

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	ErrMaxInRatio = errors.New("MAX_IN_RATIO")

	_MAX_IN_RATIO = uint256.NewInt(0.3e18)
)

var WeightedMath *weightedMath

type weightedMath struct {
}

func init() {
	WeightedMath = &weightedMath{}
}

// https://etherscan.io/address/0x00612eb4f312eb6ace7acc8781196601078ae339#code#F8#L78
func (l *weightedMath) CalcOutGivenIn(
	poolTypeVer int,
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

	power, err := FixedPoint.PowUp(poolTypeVer, base, exponent)
	if err != nil {
		return nil, err
	}

	return FixedPoint.MulDown(balanceOut, FixedPoint.Complement(power))
}
