package balancerweighted

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/d3bc956f837eeb54614c64876791fafd90b4740e/contracts/lib/math/Math.sol#L62
func mul(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Mul(a, b)
}

// // Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/d3bc956f837eeb54614c64876791fafd90b4740e/contracts/lib/math/Math.sol#L68
func mathDivDown(a *big.Int, b *big.Int) *big.Int {
	if b.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}

	return new(big.Int).Div(a, b)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/bb3658b700d1e72fd66b2c367aca326c82e6ff0f/contracts/pools/weighted/WeightedPool2Tokens.sol#L1043
func _upscale(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return mul(amount, scalingFactor)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/bb3658b700d1e72fd66b2c367aca326c82e6ff0f/contracts/pools/weighted/WeightedPool2Tokens.sol#L1077
//func _downscaleUp(amount *big.Int, scalingFactor *big.Int) *big.Int {
//	return divUp(amount, scalingFactor)
//}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/bb3658b700d1e72fd66b2c367aca326c82e6ff0f/contracts/pools/weighted/WeightedPool2Tokens.sol#L1060
func _downscaleDown(amount *big.Int, scalingFactor *big.Int) *big.Int {
	return mathDivDown(amount, scalingFactor)
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/bb3658b700d1e72fd66b2c367aca326c82e6ff0f/contracts/pools/weighted/WeightedPool2Tokens.sol#L1022
func _computeScalingFactor(tokenDecimals uint) *big.Int {
	var decimalsDiff = 18 - tokenDecimals
	return constant.TenPowInt(uint8(decimalsDiff))
}

// Solidity code: https://github.com/balancer-labs/balancer-v2-monorepo/blob/369af964a657af65ba9343178f018cce57a41442/contracts/pools/weighted/WeightedMath.sol#L69
func calcOutGivenIn(
	balanceIn *big.Int,
	weightIn *big.Int,
	balanceOut *big.Int,
	weightOut *big.Int,
	amountIn *big.Int,
) *big.Int {
	/**********************************************************************************************
	// outGivenIn                                                                                //
	// aO = amountOut                                                                            //
	// bO = balanceOut                                                                           //
	// bI = balanceIn              /      /            bI             \    (wI / wO) \           //
	// aI = amountIn    aO = bO * |  1 - | --------------------------  | ^            |          //
	// wI = weightIn               \      \       ( bI + aI )         /              /           //
	// wO = weightOut                                                                            //
	**********************************************************************************************/

	// Amount out, so we round down overall.

	// The multiplication rounds down, and the subtrahend (power) rounds up (so the base rounds up too).
	// Because bI / (bI + aI) <= 1, the exponent rounds down.

	var denominator = new(big.Int).Add(balanceIn, amountIn)
	var base = divUp(balanceIn, denominator)
	var exponent = divDown(weightIn, weightOut)
	var power = powUp(base, exponent)

	return mulDown(balanceOut, complement(power))
}
