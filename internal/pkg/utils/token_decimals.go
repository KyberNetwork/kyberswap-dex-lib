package utils

import (
	"math/big"
)

const (
	MIN_TOKEN_DECIMAL = 0
	MAX_TOKEN_DECIMAL = 50 // currently we only have maximum 18 decimals, so 50 should be enough
)

var TenPowDecimals [MAX_TOKEN_DECIMAL + 1]*big.Int

func init() {
	ten := big.NewInt(10)
	for decimals := MIN_TOKEN_DECIMAL; decimals <= MAX_TOKEN_DECIMAL; decimals++ {
		TenPowDecimals[decimals] = new(big.Int).Exp(
			ten,
			big.NewInt(int64(decimals)),
			nil,
		)
	}
}

func TenPowDecimalsFloat(decimals int) *big.Float {
	if decimals < MIN_TOKEN_DECIMAL || decimals > MAX_TOKEN_DECIMAL {
		return nil
	}
	return new(big.Float).SetInt(TenPowDecimals[decimals])
}
