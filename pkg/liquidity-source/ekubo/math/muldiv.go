package math

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func MulDivOverflow(x, y, d *big.Int, roundUp bool) (*big.Int, error) {
	if d.Sign() == 0 {
		return nil, ErrDivZero
	}

	temp := new(big.Int).Mul(x, y)
	res := new(big.Int).Div(temp, d)

	if roundUp && temp.Mod(temp, d).Sign() > 0 {
		res.Add(res, bignum.One)
	}

	if res.BitLen() > 256 {
		return nil, ErrMulDivOverflow
	}

	return res, nil
}

func div(x, y *big.Int, roundUp bool) (*big.Int, error) {
	if y.Sign() == 0 {
		return nil, ErrDivZero
	}

	quotient, remainder := new(big.Int).DivMod(x, y, new(big.Int))
	if roundUp && remainder.Sign() != 0 {
		quotient.Add(quotient, bignum.One)
	}

	return quotient, nil
}
