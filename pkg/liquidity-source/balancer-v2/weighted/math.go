package weighted

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
)

var (
	ErrMaxInRatio = errors.New("MAX_IN_RATIO")
)

var WeightedMath *weightedMath

type weightedMath struct {
}

func init() {
	WeightedMath = &weightedMath{}
}

func (l *weightedMath) _calcOutGivenIn(
	balanceIn *uint256.Int,
	weightIn *uint256.Int,
	balanceOut *uint256.Int,
	weightOut *uint256.Int,
	amountIn *uint256.Int,
) (*uint256.Int, error) {
	denominator, err := math.FixedPoint.Add(balanceIn, amountIn)
	if err != nil {
		return nil, err
	}

	base, err := math.FixedPoint.DivUp(balanceIn, denominator)
	if err != nil {
		return nil, err
	}

	exponent, err := math.FixedPoint.DivDown(weightIn, weightOut)
	if err != nil {
		return nil, err
	}

	power, err := math.FixedPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	return math.FixedPoint.MulDown(balanceOut, math.FixedPoint.Complement(power))
}
