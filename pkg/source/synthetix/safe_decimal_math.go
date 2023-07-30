package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/Synthetixio/synthetix/blob/26cf098d2c603ef2ddcd7bc3a81a9a3bbff48e90/contracts/SafeDecimalMath.sol

var (
	decimals = uint8(18)
	UNIT     = bignumber.TenPowInt(decimals)
)

func unit() *big.Int {
	return UNIT
}

/**
 * @return The result of multiplying x and y, interpreting the operands as fixed-point
 * decimals.
 *
 * @dev A unit factor is divided out after the product of x and y is evaluated,
 * so that product must be less than 2**256. As this is an integer division,
 * the internal division always rounds down. This helps save on gas. Rounding
 * is more expensive on gas.
 */
func multiplyDecimal(x, y *big.Int) *big.Int {
	return new(big.Int).Div(new(big.Int).Mul(x, y), UNIT)
}

/**
 * @return The result of safely multiplying x and y, interpreting the operands
 * as fixed-point decimals of the specified precision unit.
 *
 * @dev The operands should be in the form of a the specified unit factor which will be
 * divided out after the product of x and y is evaluated, so that product must be
 * less than 2**256.
 *
 * Unlike multiplyDecimal, this function rounds the result to the nearest increment.
 * Rounding is useful when you need to retain fidelity for small decimal numbers
 * (eg. small fractions or percentages).
 */
func _multiplyDecimalRound(x, y, precisionUnit *big.Int) *big.Int {
	quotientTimesTen := new(big.Int).Div(new(big.Int).Mul(x, y), new(big.Int).Div(precisionUnit, big.NewInt(10)))

	if new(big.Int).Mod(quotientTimesTen, big.NewInt(10)).Cmp(big.NewInt(5)) >= 0 {
		quotientTimesTen = new(big.Int).Add(quotientTimesTen, big.NewInt(10))
	}

	return new(big.Int).Div(quotientTimesTen, big.NewInt(10))
}

/**
 * @return The result of safely multiplying x and y, interpreting the operands
 * as fixed-point decimals of a standard unit.
 *
 * @dev The operands should be in the standard unit factor which will be
 * divided out after the product of x and y is evaluated, so that product must be
 * less than 2**256.
 *
 * Unlike multiplyDecimal, this function rounds the result to the nearest increment.
 * Rounding is useful when you need to retain fidelity for small decimal numbers
 * (eg. small fractions or percentages).
 */
func multiplyDecimalRound(x, y *big.Int) *big.Int {
	return _multiplyDecimalRound(x, y, UNIT)
}

/**
 * @return The result of safely dividing x and y. The return value is a high
 * precision decimal.
 *
 * @dev y is divided after the product of x and the standard precision unit
 * is evaluated, so the product of x and UNIT must be less than 2**256. As
 * this is an integer division, the result is always rounded down.
 * This helps save on gas. Rounding is more expensive on gas.
 */
func divideDecimal(x, y *big.Int) *big.Int {
	/* Reintroduce the UNIT factor that will be divided out by y. */
	return new(big.Int).Div(new(big.Int).Mul(x, UNIT), y)
}

/**
 * @return The result of safely dividing x and y. The return value is as a rounded
 * decimal in the precision unit specified in the parameter.
 *
 * @dev y is divided after the product of x and the specified precision unit
 * is evaluated, so the product of x and the specified precision unit must
 * be less than 2**256. The result is rounded to the nearest increment.
 */
func _divideDecimalRound(x, y, precisionUnit *big.Int) *big.Int {
	resultTimesTen := new(big.Int).Div(new(big.Int).Mul(x, new(big.Int).Mul(precisionUnit, big.NewInt(10))), y)

	if new(big.Int).Mod(resultTimesTen, big.NewInt(10)).Cmp(big.NewInt(5)) >= 0 {
		resultTimesTen = new(big.Int).Add(resultTimesTen, big.NewInt(10))
	}

	return new(big.Int).Div(resultTimesTen, big.NewInt(10))
}

func divideDecimalRound(x, y *big.Int) *big.Int {
	return _divideDecimalRound(x, y, UNIT)
}
