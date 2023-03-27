package utils

import "math/big"

func SafeCastBigIntToString(num *big.Int) string {
	if num == nil {
		return EmptyString
	}

	return num.String()
}

func SafeCastBigIntToInt64(num *big.Int) int64 {
	if num == nil {
		return Zero
	}

	return num.Int64()
}
