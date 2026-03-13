package alphix

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Q96 is 2^96, used in fixed-point sqrt price math.
var Q96 = new(uint256.Int).Lsh(uint256.NewInt(1), 96)

// getLiquidityForAmounts computes the maximum liquidity that can be minted
// at [sqrtPriceLower, sqrtPriceUpper] given available amounts.
// This mirrors Uniswap's LiquidityAmounts.getLiquidityForAmounts.
func getLiquidityForAmounts(
	sqrtPriceX96, sqrtPriceLowerX96, sqrtPriceUpperX96 *uint256.Int,
	amount0, amount1 *uint256.Int,
) *uint256.Int {
	if sqrtPriceLowerX96.Cmp(sqrtPriceUpperX96) >= 0 {
		return uint256.NewInt(0)
	}

	if sqrtPriceX96.Cmp(sqrtPriceLowerX96) <= 0 {
		return getLiquidityForAmount0(sqrtPriceLowerX96, sqrtPriceUpperX96, amount0)
	}
	if sqrtPriceX96.Cmp(sqrtPriceUpperX96) >= 0 {
		return getLiquidityForAmount1(sqrtPriceLowerX96, sqrtPriceUpperX96, amount1)
	}
	// Both tokens — take the minimum
	liq0 := getLiquidityForAmount0(sqrtPriceX96, sqrtPriceUpperX96, amount0)
	liq1 := getLiquidityForAmount1(sqrtPriceLowerX96, sqrtPriceX96, amount1)
	if liq0.Cmp(liq1) < 0 {
		return liq0
	}
	return liq1
}

// getLiquidityForAmount0 computes liquidity from token0.
// L = amount0 * sqrtA * sqrtB / ((sqrtB - sqrtA) * Q96)
//
// We use big.Int for the intermediate multiplication to avoid uint256 overflow,
// since sqrtA * sqrtB can be up to ~2^192.
func getLiquidityForAmount0(sqrtPriceAX96, sqrtPriceBX96, amount0 *uint256.Int) *uint256.Int {
	if sqrtPriceAX96.Cmp(sqrtPriceBX96) > 0 {
		sqrtPriceAX96, sqrtPriceBX96 = sqrtPriceBX96, sqrtPriceAX96
	}
	diff := new(uint256.Int).Sub(sqrtPriceBX96, sqrtPriceAX96)
	if diff.IsZero() {
		return uint256.NewInt(0)
	}

	// Use big.Int to avoid overflow: numerator = amount0 * sqrtA * sqrtB
	a0Big := amount0.ToBig()
	sqrtABig := sqrtPriceAX96.ToBig()
	sqrtBBig := sqrtPriceBX96.ToBig()
	diffBig := diff.ToBig()
	q96Big := Q96.ToBig()

	numerator := new(big.Int).Mul(a0Big, sqrtABig)
	numerator.Mul(numerator, sqrtBBig)

	// denominator = diff * Q96
	denominator := new(big.Int).Mul(diffBig, q96Big)

	result := numerator.Div(numerator, denominator)
	liq, overflow := uint256.FromBig(result)
	if overflow {
		return uint256.NewInt(0)
	}
	return liq
}

// getLiquidityForAmount1 computes liquidity from token1.
// L = amount1 * Q96 / (sqrtB - sqrtA)
func getLiquidityForAmount1(sqrtPriceAX96, sqrtPriceBX96, amount1 *uint256.Int) *uint256.Int {
	if sqrtPriceAX96.Cmp(sqrtPriceBX96) > 0 {
		sqrtPriceAX96, sqrtPriceBX96 = sqrtPriceBX96, sqrtPriceAX96
	}
	diff := new(uint256.Int).Sub(sqrtPriceBX96, sqrtPriceAX96)
	if diff.IsZero() {
		return uint256.NewInt(0)
	}
	numerator := new(uint256.Int).Mul(amount1, Q96)
	return numerator.Div(numerator, diff)
}

// computeJitSwap simulates a swap against the JIT concentrated liquidity position.
// Returns (deltaSpecified, deltaUnspecified) representing the portion of the swap
// that the JIT position fills.
//
// The V3 simulator handles the rest against non-rehypothecated tick liquidity.
func computeJitSwap(
	zeroForOne, exactIn bool,
	amountSpecified *big.Int,
	sqrtPriceX96, sqrtPriceLowerX96, sqrtPriceUpperX96 *uint256.Int,
	liquidity *uint256.Int,
	swapFee uniswapv4.FeeAmount,
) (deltaSpecified, deltaUnspecified *big.Int) {
	// Determine the price limit for the JIT swap
	sqrtPriceTargetX96 := sqrtPriceLowerX96
	if !zeroForOne {
		sqrtPriceTargetX96 = sqrtPriceUpperX96
	}

	// If price is outside JIT range, JIT cannot contribute
	if zeroForOne && sqrtPriceX96.Cmp(sqrtPriceLowerX96) <= 0 {
		return big.NewInt(0), big.NewInt(0)
	}
	if !zeroForOne && sqrtPriceX96.Cmp(sqrtPriceUpperX96) >= 0 {
		return big.NewInt(0), big.NewInt(0)
	}

	// Convert amountSpecified to int256 for the V3 SDK
	// For exactIn: amountRemaining is positive
	// For exactOut: amountRemaining is negative
	amountRemainingI256 := int256.NewInt(0)
	amountRemainingI256.SetFromBig(new(big.Int).Abs(amountSpecified))
	if !exactIn {
		amountRemainingI256.Neg(amountRemainingI256)
	}

	var sqrtPriceNextX96, amountIn, amountOut, feeAmount uint256.Int
	err := v3Utils.ComputeSwapStep(
		sqrtPriceX96,
		sqrtPriceTargetX96,
		liquidity,
		amountRemainingI256,
		swapFee,
		&sqrtPriceNextX96, &amountIn, &amountOut, &feeAmount,
	)
	if err != nil {
		return big.NewInt(0), big.NewInt(0)
	}

	// deltaSpecified: amount consumed from the input (fee included)
	totalIn := new(uint256.Int).Add(&amountIn, &feeAmount)
	deltaSpecified = totalIn.ToBig()

	// deltaUnspecified: negative output (convention: negative = tokens going to swapper)
	deltaUnspecified = new(big.Int).Neg(amountOut.ToBig())

	return deltaSpecified, deltaUnspecified
}
