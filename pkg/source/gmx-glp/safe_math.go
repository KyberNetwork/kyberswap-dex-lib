package gmxglp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

func mul(a, b *big.Int) (*big.Int, error) {
	if a.Cmp(bignumber.ZeroBI) == 0 {
		return new(big.Int).Set(bignumber.ZeroBI), nil
	}
	c := new(big.Int).Mul(a, b)
	if new(big.Int).Div(c, a).Cmp(b) != 0 {
		return nil, ErrSafeMathMulOverflow
	}

	return c, nil
}

func div(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrSafeMathDivZero
	}
	c := new(big.Int).Div(a, b)

	return c, nil
}

func sub(a, b *big.Int) (*big.Int, error) {
	if b.Cmp(a) > 0 {
		return nil, ErrSafeMathSubOverflow
	}

	return new(big.Int).Sub(a, b), nil
}
