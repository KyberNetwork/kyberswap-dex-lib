package balancerweighted

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// MAX_POW_RELATIVE_ERROR keeps it snake-case so that it looks like Solidity code, easier to compare
var MAX_POW_RELATIVE_ERROR = bignumber.NewBig10("10000")

// Solidity code:
// https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L45
func mulDown(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	return new(big.Int).Div(ret, bignumber.BONE)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L52
func mulUp(a *big.Int, b *big.Int) *big.Int {
	var product = new(big.Int).Mul(a, b)
	if product.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(product, bignumber.One), bignumber.BONE), bignumber.One)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L69
func divDown(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}
	var aInflated = new(big.Int).Mul(a, bignumber.BONE)

	return new(big.Int).Div(aInflated, b)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L82
func divUp(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}
	var aInflated = new(big.Int).Mul(a, bignumber.BONE)

	// The traditional divUp formula is:
	// divUp(x, y) := (x + y - 1) / y
	// To avoid intermediate overflow in the addition, we distribute the division and get:
	// divUp(x, y) := (x - 1) / y + 1
	// Note that this requires x != 0, which we already tested for.
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(aInflated, bignumber.One), b), bignumber.One)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L120
func powUp(x *big.Int, y *big.Int) *big.Int {
	var raw = pow(x, y)
	var maxError = new(big.Int).Add(mulUp(raw, MAX_POW_RELATIVE_ERROR), bignumber.One)
	return new(big.Int).Add(raw, maxError)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L133
func complement(x *big.Int) *big.Int {
	if x.Cmp(bignumber.BONE) < 0 {
		return new(big.Int).Sub(bignumber.BONE, x)
	}
	return bignumber.ZeroBI
}
