package smardex

import "math/big"

var APPROX_PRECISION = big.NewInt(1)
var APPROX_PRECISION_BASE = big.NewInt(1_000_000)

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

func sqrt(value *big.Int) *big.Int {
	if value.Cmp(big.NewInt(0)) == 0 {
		return value
	}
	result := new(big.Int).Lsh(big.NewInt(1), log2(value)/2)
	tmp := new(big.Int)
	result.Rsh(tmp.Div(value, result).Add(tmp, result), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	result.Rsh(tmp.Div(value, result).Add(result, tmp), 1)
	tmp = new(big.Int).Div(value, result)
	if result.Cmp(tmp) == -1 {
		return result
	}
	return tmp
}

func log2(value *big.Int) uint {
	var result uint = 0
	zero := big.NewInt(0)
	comparator := new(big.Int)
	tempValue := new(big.Int).Set(value)
	if comparator.Rsh(tempValue, 128).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 128
	}
	if comparator.Rsh(tempValue, 64).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 64
	}
	if comparator.Rsh(tempValue, 32).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 32
	}
	if comparator.Rsh(tempValue, 16).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 16
	}
	if comparator.Rsh(tempValue, 8).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 8
	}
	if comparator.Rsh(tempValue, 4).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 4
	}
	if comparator.Rsh(tempValue, 2).Cmp(zero) == 1 {
		tempValue.Set(comparator)
		result += 2
	}
	if comparator.Rsh(tempValue, 1).Cmp(zero) == 1 {
		result += 1
	}

	return result
}
