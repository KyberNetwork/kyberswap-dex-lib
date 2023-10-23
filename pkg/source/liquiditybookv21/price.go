package liquiditybookv21

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getPriceFromID(id uint32, binStep uint32) *big.Int {
	base := getBase(binStep)
	exponent := getExponent(id)
	return pow(base, exponent)
}

func getBase(binStep uint32) *big.Int {
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

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func pow(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		invert bool
		absY   *big.Int
	)

	if y.Cmp(bignumber.ZeroBI) == 0 {
		return scale, nil
	}

	absY = new(big.Int).Abs(y)
	if y.Sign() < 0 {
		invert = true
	}

	u, _ := new(big.Int).SetString("100000", 16)
	if absY.Cmp(u) < 0 {
		v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
		squared := x
		if x.Cmp(v) > 0 {
			squared = new(big.Int).Div(
				
			)
		}
	}

	return nil
}
