package smardex

import "math/big"

var APPROX_EQ_PRECISION = big.NewInt(1)
var APPROX_EQ_BASE_PRECISION = big.NewInt(1000000)

func isZero(a *big.Int) bool {
	return big.NewInt(0).Cmp(a) == 0
}

/**
 * @notice check if 2 ratio are approximately equal: _xNum _/ xDen ~= _yNum / _yDen
 * @param _xNum numerator of the first ratio to compare
 * @param _xDen denominator of the first ratio to compare
 * @param _yNum numerator of the second ratio to compare
 * @param _yDen denominator of the second ratio to compare
 * @return true if ratio are approximately equal, false otherwise
 */
func ratioApproxEq(xNum *big.Int, xDen *big.Int, yNum *big.Int, yDen *big.Int) bool {
	return approxEq(new(big.Int).Mul(xNum, yDen), new(big.Int).Mul(xDen, yNum))
}

/**
 * @notice check if 2 numbers are approximately equal, using APPROX_PRECISION
 * @param _x number to compare
 * @param _y number to compare
 * @return true if numbers are approximately equal, false otherwise
 */
func approxEq(x *big.Int, y *big.Int) bool {
	res := big.NewInt(0)
	if x.Cmp(y) == 1 {
		return x.Cmp(res.Mul(y, APPROX_EQ_PRECISION).Div(res, APPROX_EQ_BASE_PRECISION).Add(res, y)) == -1
	}
	return y.Cmp(res.Mul(x, APPROX_EQ_PRECISION).Div(res, APPROX_EQ_BASE_PRECISION).Add(res, x)) == -1
}
