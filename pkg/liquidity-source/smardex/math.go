package smardex

import (
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

var (
	APPROX_EQ_PRECISION      = u256.U1
	APPROX_EQ_BASE_PRECISION = uint256.NewInt(1000000)
)

/**
 * @notice check if 2 ratio are approximately equal: _xNum _/ xDen ~= _yNum / _yDen
 * @param _xNum numerator of the first ratio to compare
 * @param _xDen denominator of the first ratio to compare
 * @param _yNum numerator of the second ratio to compare
 * @param _yDen denominator of the second ratio to compare
 * @return true if ratio are approximately equal, false otherwise
 */
func ratioApproxEq(xNum *uint256.Int, xDen *uint256.Int, yNum *uint256.Int, yDen *uint256.Int) bool {
	return approxEq(new(uint256.Int).Mul(xNum, yDen), new(uint256.Int).Mul(xDen, yNum))
}

/**
 * @notice check if 2 numbers are approximately equal, using APPROX_PRECISION
 * @param _x number to compare
 * @param _y number to compare
 * @return true if numbers are approximately equal, false otherwise
 */
func approxEq(x *uint256.Int, y *uint256.Int) bool {
	cmp := x.Cmp(y)
	if cmp == 0 {
		return true
	}

	temp := new(uint256.Int)
	if cmp == 1 {
		temp.MulDivOverflow(y, APPROX_EQ_PRECISION, APPROX_EQ_BASE_PRECISION)
		return x.Lt(temp.Add(temp, y))
	}

	temp.MulDivOverflow(x, APPROX_EQ_PRECISION, APPROX_EQ_BASE_PRECISION)
	return y.Lt(temp.Add(temp, x))
}
