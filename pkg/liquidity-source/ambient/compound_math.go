package ambient

import "math/big"

/* @notice Inflates a starting value by a cumulative growth rate.
* @dev    Rounds down from the real value. Result is capped at max(uint128).
* @param seed The pre-inflated starting value as unsigned integer
* @param growth Cumulative growth rate as Q16.48 fixed-point
* @return The ending value = seed * (1 + growth). Rounded down to nearest
*         integer value */
var (
	maxUint128, _ = new(big.Int).SetString("340282366920938463463374607431768211455", 10)
)

func inflateLiqSeed(seed *big.Int, growth uint64) *big.Int {
	// 	 uint256 ONE = FixedPoint.Q48;
	// 	 uint256 num = uint256(seed) * uint256(ONE + growth); // Guaranteed to fit in 256
	num := new(big.Int).Set(seed)
	num.Mul(num, new(big.Int).Add(fixedPointQ48, new(big.Int).SetUint64(growth)))

	// uint256 inflated = num >> 48; // De-scale by the 48-bit growth precision;
	inflated := num.Rsh(num, 48)

	// 	 if (inflated > type(uint128).max) { return type(uint128).max; }
	// 	 return uint128(inflated);
	// 	 }
	if inflated.Cmp(maxUint128) == 1 {
		return new(big.Int).Set(maxUint128)
	}

	return inflated
}
