package liquiditybookv21

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getPriceFromIDBackup(id uint32, binStep uint16) (*big.Int, error) {
	base := getBaseBackup(binStep)
	exponent := getExponentBackup(id)
	return powBackup(base, exponent)
}

func getBaseBackup(binStep uint16) *big.Int {
	u := new(big.Int).Lsh(big.NewInt(int64(binStep)), scaleOffset)
	return new(big.Int).Add(scale, new(big.Int).Div(u, big.NewInt(basisPointMax)))
}

func getExponentBackup(id uint32) *big.Int {
	return new(big.Int).Sub(big.NewInt(int64(id)), big.NewInt(realIDShift))
}

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func powBackup(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		invert bool
		absY   *big.Int
		result = big.NewInt(0)
	)

	if y.Cmp(integer.Zero()) == 0 {
		return scale, nil
	}

	absY = new(big.Int).Abs(y)
	if y.Sign() < 0 {
		invert = !invert
	}

	u, _ := new(big.Int).SetString("100000", 16)
	if absY.Cmp(u) < 0 {
		result = scale

		squared := x
		v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
		if x.Cmp(v) > 0 {
			squared = new(big.Int).Div(bignumber.MAX_UINT_256, squared)

			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			and := new(big.Int).And(absY, big.NewInt(int64(i)))
			if and.Cmp(integer.Zero()) != 0 {
				result = new(big.Int).Rsh(
					new(big.Int).Mul(result, squared),
					128,
				)
			}
			if i < 0x80000 {
				squared = new(big.Int).Rsh(
					new(big.Int).Mul(squared, squared),
					128,
				)
			}
		}
	}

	if result.Cmp(integer.Zero()) == 0 {
		return nil, ErrPowUnderflow
	}

	if invert {
		v := new(big.Int).Sub(new(big.Int).Lsh(integer.One(), 256), integer.One())
		result = new(big.Int).Div(v, result)
	}

	return result, nil
}
