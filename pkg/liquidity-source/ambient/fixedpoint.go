package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	Q48  = new(big.Int).Lsh(bignum.One, 48)
	Q128 = bignum.B2Pow128

	mask128 = bignum.MaxUint128
)

func MulQ64(x, y *big.Int) *big.Int {
	result := new(big.Int).Mul(x, y)
	return result.Rsh(result, 64)
}

func DivQ64(x, y *big.Int) *big.Int {
	num := new(big.Int).Lsh(x, 64)
	return num.Div(num, y)
}

func MulQ48(x *big.Int, y uint64) *big.Int {
	result := new(big.Int).Mul(x, new(big.Int).SetUint64(y))
	return result.Rsh(result, 48)
}

func RecipQ64(x *big.Int) *big.Int {
	return new(big.Int).Div(Q128, x)
}
