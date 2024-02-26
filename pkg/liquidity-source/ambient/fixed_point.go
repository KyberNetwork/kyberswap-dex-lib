package ambient

import "math/big"

var (
	fixedPointQ48, _ = new(big.Int).SetString("0x1000000000000", 16)
)

// /* @notice Multiplies two Q64.64 numbers by each other. */
func mulQ64(x, y *big.Int) *big.Int {
	// return uint192((uint256(x) * uint256(y)) >> 64);
	z := new(big.Int).Mul(x, y)
	z.Rsh(z, 64)

	return z
}

// /* @notice Divides one Q64.64 number by another. */
func divQ64(x, y *big.Int) *big.Int {
	// return (uint192(x) << 64) / y;
	z := new(big.Int).Set(x)
	z.Lsh(z, 64)
	z.Div(z, y)

	return z
}

/* @notice Multiplies a Q64.64 by a Q16.48. */
func mulQ48(x *big.Int, y *big.Int) *big.Int {
	//     return uint144((uint256(x) * uint256(y)) >> 48);
	tmp := new(big.Int).Mul(x, y)
	tmp.Rsh(tmp, 48)

	return tmp
}
