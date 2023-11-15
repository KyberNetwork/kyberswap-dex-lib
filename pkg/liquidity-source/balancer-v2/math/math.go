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

	if !(a.Cmp(number.Zero) == 0 || new(uint256.Int).Div(c, a).Cmp(b) == 0) {
		return nil, ErrMulOverflow
	}

	return c, nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/Math.sol#L97
func (m *math) DivDown(a *uint256.Int, b *uint256.Int) (*uint256.Int, error) {
	if b.Cmp(number.Zero) == 0 {
		return nil, ErrZeroDivision
	}

	return new(uint256.Int).Div(a, b), nil
}
