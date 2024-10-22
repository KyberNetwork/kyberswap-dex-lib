package meth

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	// 2^256 - 1
	maxUint256 = new(uint256.Int).SetAllOne()
)

func mulDiv(x, y, denominator *uint256.Int) (*uint256.Int, error) {
	if denominator.IsZero() {
		return nil, number.ErrDivByZero
	}

	mm := new(uint256.Int).MulMod(x, y, maxUint256)
	prod0 := new(uint256.Int).Mul(x, y)

	prod1 := new(uint256.Int).Sub(mm, prod0)
	if mm.Cmp(prod0) < 0 {
		prod1.Sub(mm, prod0).Sub(prod1, uint256.NewInt(1))
	} else {
		prod1.Sub(mm, prod0)
	}

	if prod1.IsZero() {
		return new(uint256.Int).Div(prod0, denominator), nil
	}

	if denominator.Cmp(prod1) <= 0 {
		return nil, number.ErrOverflow
	}

	// 512 by 256 division
	// this code below will not execute because the minDeposit and maxDeposit conditions prevent it
	remainder := new(uint256.Int).MulMod(x, y, denominator)
	if remainder.Cmp(prod0) > 0 {
		prod1.Sub(prod1, uint256.NewInt(1))
	}

	prod0.Sub(prod0, remainder)

	twos := new(uint256.Int).And(denominator, new(uint256.Int).Sub(uint256.NewInt(0), denominator))
	denominator.Div(denominator, twos)
	prod0.Div(prod0, twos)

	if twos.IsZero() {
		twos.SetUint64(1)
	} else {
		twos = new(uint256.Int).Div(new(uint256.Int).Sub(uint256.NewInt(0), twos), twos).Add(twos, uint256.NewInt(1))
	}

	prod0.Or(prod0, new(uint256.Int).Mul(prod1, twos))

	inverse := new(uint256.Int).Xor(new(uint256.Int).Mul(uint256.NewInt(3), denominator), uint256.NewInt(2))
	for i := 0; i < 6; i++ {
		inverse.Mul(inverse, new(uint256.Int).Sub(uint256.NewInt(2), new(uint256.Int).Mul(denominator, inverse)))
	}

	result := new(uint256.Int).Mul(prod0, inverse)
	return result, nil
}
