package math

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var Math *math

type math struct{}

func init() {
	Math = &math{}
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/Math.sol#L83
func (m *math) Mul(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	c := new(uint256.Int).Mul(a, b)

	if !(a.IsZero() || new(uint256.Int).Div(c, a).Eq(b)) {
		return nil, ErrMulOverflow
	}

	return c, nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/Math.sol#L97
func (m *math) DivDown(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	return new(uint256.Int).Div(a, b), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/solidity-utils/contracts/math/Math.sol#L102
func (m *math) DivUp(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrZeroDivision
	}

	if a.IsZero() {
		return number.Zero, nil
	}

	c := new(uint256.Int).Add(
		number.Number_1,
		new(uint256.Int).Div(
			new(uint256.Int).Sub(a, number.Number_1),
			b,
		),
	)

	return c, nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/master/pkg/solidity-utils/contracts/math/Math.sol#L89
func (m *math) Div(a *uint256.Int, b *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if roundUp {
		return m.DivUp(a, b)
	}
	return m.DivDown(a, b)
}

func (m *math) Min(a *uint256.Int, b *uint256.Int) *uint256.Int {
	if a.Lt(b) {
		return a
	}
	return b
}

func (m *math) Max(a *uint256.Int, b *uint256.Int) *uint256.Int {
	if a.Gt(b) {
		return a
	}
	return b
}
