package liquiditybookv21

import (
	"math/big"
)

func getPriceFromID(id uint32, binStep uint16) (*big.Int, error) {
	base := getBase(binStep)
	exponent := getExponent(id)
	return pow(base, exponent)
}

func getBase(binStep uint16) *big.Int {
	u := new(big.Int).Lsh(big.NewInt(int64(binStep)), scaleOffset)
	return new(big.Int).Add(
		scale,
		new(big.Int).Div(
			u,
			big.NewInt(basisPointMax),
		),
	)
}

func getExponent(id uint32) *big.Int {
	return new(big.Int).Sub(big.NewInt(int64(id)), big.NewInt(realIDShift))
}
