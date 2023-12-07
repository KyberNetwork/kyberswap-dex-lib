package math

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var ASM *asm

type asm struct{}

func (*asm) Not(u *uint256.Int) *uint256.Int {
	return new(uint256.Int).Not(u)
}

func (*asm) MulMod(x, y, z *uint256.Int) *uint256.Int {
	return new(uint256.Int).MulMod(x, y, z)
}

func (*asm) Lt(x, y *uint256.Int) *uint256.Int {
	if x.Lt(y) {
		return number.Number_1
	}
	return number.Zero
}

func (*asm) Gt(x, y *uint256.Int) *uint256.Int {
	if x.Gt(y) {
		return number.Number_1
	}
	return number.Zero
}

func (*asm) Mul(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Mul(x, y)
}

func (*asm) Sub(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sub(x, y)
}

func (*asm) Div(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(x, y)
}

func (*asm) Or(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Or(x, y)
}

func (*asm) Add(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Add(x, y)
}

func (*asm) Mod(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Mod(x, y)
}

func (*asm) Eq(x, y *uint256.Int) *uint256.Int {
	if x.Eq(y) {
		return number.Number_1
	}
	return number.Zero
}

func (*asm) IsZero(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return number.Number_1
	}
	return number.Zero
}
