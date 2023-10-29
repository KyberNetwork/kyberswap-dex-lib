package smardex

import "math/big"

var APPROX_PRECISION = big.NewInt(1)
var APPROX_PRECISION_BASE = big.NewInt(1_000_000)

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

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
	if x.Cmp(y) > 1 {
		return x.Cmp(new(big.Int).Add(y, new(big.Int).Div(new(big.Int).Mul(y, APPROX_PRECISION), APPROX_PRECISION_BASE))) < 0
	} else {
		return y.Cmp(new(big.Int).Add(x, new(big.Int).Div(new(big.Int).Mul(x, APPROX_PRECISION), APPROX_PRECISION_BASE))) < 0
	}
}
