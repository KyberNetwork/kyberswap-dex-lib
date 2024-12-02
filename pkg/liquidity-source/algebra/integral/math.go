package integral

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func unsafeDivRoundingUp(x, y *big.Int) (*big.Int, error) {
	if y.Sign() == 0 {
		return nil, errors.New("division by zero")
	}

	quotient := new(big.Int).Div(x, y)

	remainder := new(big.Int).Mod(x, y)
	if remainder.Sign() > 0 {
		quotient.Add(quotient, bignumber.One)
	}

	return quotient, nil
}

func mulDiv(a, b, denominator *big.Int) (*big.Int, error) {
	if denominator.Sign() == 0 {
		return nil, errors.New("denominator must be greater than zero")
	}

	prod0 := new(big.Int).Mul(a, b)
	prod1 := new(big.Int)

	mulMod := new(big.Int).Mod(new(big.Int).Mul(a, b), new(big.Int).SetUint64(^uint64(0)))
	prod1 = new(big.Int).Sub(new(big.Int).Sub(mulMod, prod0), bignumber.ZeroBI)
	if mulMod.Cmp(prod0) < 0 {
		prod1.Sub(prod1, bignumber.One)
	}

	if denominator.Cmp(prod1) <= 0 {
		return nil, errors.New("denominator must be greater than prod1")
	}

	if prod1.Sign() == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	remainder := new(big.Int).Mod(new(big.Int).Mul(a, b), denominator)
	prod1.Sub(prod1, bignumber.ZeroBI)
	if remainder.Cmp(prod0) > 0 {
		prod1.Sub(prod1, bignumber.One)
	}
	prod0.Sub(prod0, remainder)

	twos := new(big.Int).And(new(big.Int).Neg(denominator), denominator)
	denominator.Div(denominator, twos)

	prod0.Div(prod0, twos)

	twosComplement := new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(bignumber.ZeroBI, twos), twos), bignumber.One)
	prod0.Or(prod0, new(big.Int).Mul(prod1, twosComplement))

	inv := new(big.Int).Mul(denominator, bignumber.Three)
	inv.Xor(inv, bignumber.Two)
	for i := 0; i < 6; i++ {
		inv.Mul(inv, new(big.Int).Sub(bignumber.Two, new(big.Int).Mul(denominator, inv)))
	}

	result := new(big.Int).Mul(prod0, inv)
	result.Mod(result, new(big.Int).Lsh(bignumber.One, 256))

	return result, nil
}
