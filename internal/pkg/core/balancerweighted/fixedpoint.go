package balancerweighted

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

// MAX_POW_RELATIVE_ERROR keeps it snake-case so that it looks like Solidity code, easier to compare
var MAX_POW_RELATIVE_ERROR = utils.NewBig10("10000")

// Solidity code:
// https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L45
func mulDown(a *big.Int, b *big.Int) *big.Int {
	var ret = new(big.Int).Mul(a, b)
	return new(big.Int).Div(ret, constant.BONE)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L52
func mulUp(a *big.Int, b *big.Int) *big.Int {
	var product = new(big.Int).Mul(a, b)
	if product.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}

	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(product, constant.One), constant.BONE), constant.One)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L69
func divDown(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	var aInflated = new(big.Int).Mul(a, constant.BONE)

	return new(big.Int).Div(aInflated, b)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L82
func divUp(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	var aInflated = new(big.Int).Mul(a, constant.BONE)

	// The traditional divUp formula is:
	// divUp(x, y) := (x + y - 1) / y
	// To avoid intermediate overflow in the addition, we distribute the division and get:
	// divUp(x, y) := (x - 1) / y + 1
	// Note that this requires x != 0, which we already tested for.
	return new(big.Int).Add(new(big.Int).Div(new(big.Int).Sub(aInflated, constant.One), b), constant.One)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L120
func powUp(x *big.Int, y *big.Int) *big.Int {
	var raw = pow(x, y)
	var maxError = new(big.Int).Add(mulUp(raw, MAX_POW_RELATIVE_ERROR), constant.One)
	return new(big.Int).Add(raw, maxError)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/035cdf829740a60b7cd5aa7dedab413a627d69c8/contracts/lib/math/FixedPoint.sol#L133
func complement(x *big.Int) *big.Int {
	if x.Cmp(constant.BONE) < 0 {
		return new(big.Int).Sub(constant.BONE, x)
	}
	return constant.Zero
}
