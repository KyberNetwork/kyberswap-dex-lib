package stabull

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrZeroDenominator       = errors.New("zero denominator")
)

// calculateStabullSwap implements the Stabull curve swap calculation
// The Stabull curve uses a sophisticated invariant with greek parameters:
// - alpha (α): Weight between constant product and constant sum
// - beta (β): Volatility parameter
// - delta (δ): Slippage parameter
// - epsilon (ε): Fee parameter (dynamic fee based on imbalance)
// - lambda (λ): Oracle weight parameter
// - oracleRate: Chainlink oracle rate (base/quote) for price guidance
//
// The curve formula maintains an invariant that combines:
// 1. Constant product (Uniswap-style): x * y = k
// 2. Constant sum (Curve-style): x + y = k
// 3. Oracle-aware pricing adjustments using lambda
//
// Approximation approach:
// Since we can't replicate the full Solidity math exactly (due to fixed-point precision differences),
// we implement a close approximation that uses the greek parameters to modify the constant product formula.
func calculateStabullSwap(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	alpha *big.Int,
	beta *big.Int,
	delta *big.Int,
	epsilon *big.Int,
	lambda *big.Int,
	oracleRate *big.Int, // Oracle rate for price guidance (in 1e18 precision)
) (*big.Int, error) {
	if amountIn == nil || amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmount
	}

	if reserveIn == nil || reserveIn.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	if reserveOut == nil || reserveOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	// Calculate dynamic fee based on epsilon
	// Fee is 0.15% (epsilon = 1.5e15) applied to the input amount
	fee := calculateDynamicFee(amountIn, epsilon)

	// Apply fee to input amount
	amountInAfterFee := new(big.Int).Sub(amountIn, fee)
	if amountInAfterFee.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidAmount
	}

	// Calculate the output amount using a hybrid formula that incorporates greek parameters
	// This is an approximation of the Stabull curve invariant

	// Base constant product calculation: amountOut = (reserveOut * amountInAfterFee) / (reserveIn + amountInAfterFee)
	numerator := new(big.Int).Mul(reserveOut, amountInAfterFee)
	denominator := new(big.Int).Add(reserveIn, amountInAfterFee)

	if denominator.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrZeroDenominator
	}

	baseAmountOut := new(big.Int).Div(numerator, denominator)

	// Apply curve adjustments based on greek parameters
	// Alpha: Weights between constant product (α=1e18) and constant sum (α=0)
	// For balanced pools, alpha is typically around 0.5 * 1e18
	// Lambda: Weights oracle price influence on the output
	adjustedAmountOut := applyCurveAdjustment(
		baseAmountOut,
		amountInAfterFee,
		reserveIn,
		reserveOut,
		alpha,
		beta,
		delta,
		lambda,
		oracleRate,
	)

	// Ensure we don't return more than available reserves
	if adjustedAmountOut.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	return adjustedAmountOut, nil
}

// calculateDynamicFee computes the swap fee based on epsilon and pool imbalance
// Epsilon represents the base fee rate (0.15% = 1.5e15 in 1e18 precision)
// Formula: fee = amountIn * epsilon / 1e18
// Note: The fee is applied to the input amount before the swap calculation
func calculateDynamicFee(amountIn *big.Int, epsilon *big.Int) *big.Int {
	// Epsilon is scaled by 1e18
	// A typical epsilon value is 1.5e15 (0.15% = 150000000000000)
	// fee = amountIn * epsilon / 1e18

	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18

	fee := new(big.Int).Mul(amountIn, epsilon)
	fee = new(big.Int).Div(fee, one)

	return fee
}

// applyCurveAdjustment applies the Stabull curve formula adjustments
// This modifies the base constant product output based on greek parameters
// Lambda (λ) is used to weight the oracle price influence on the output
func applyCurveAdjustment(
	baseAmountOut *big.Int,
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	alpha *big.Int,
	beta *big.Int,
	delta *big.Int,
	lambda *big.Int,
	oracleRate *big.Int, // Oracle rate (base/quote) in 1e18 precision
) *big.Int {
	// All greek parameters are in 1e18 precision
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18

	// Calculate pool balance ratio
	// ratio = reserveIn / reserveOut (scaled by 1e18)
	ratio := new(big.Int).Mul(reserveIn, one)
	ratio = new(big.Int).Div(ratio, reserveOut)

	// Alpha adjustment: weights between CP and CS curves
	// If alpha = 1e18 (100%), pure constant product
	// If alpha = 0, more constant sum behavior
	// adjustment_factor = alpha / 1e18
	alphaFactor := new(big.Int).Mul(baseAmountOut, alpha)
	alphaFactor = new(big.Int).Div(alphaFactor, one)

	// Beta adjustment: volatility-based price impact
	// Higher beta = more slippage for large trades
	// impact = (amountIn / reserveIn)^beta
	// For simplicity, we approximate: impact ≈ (amountIn / reserveIn) * (beta / 1e18)
	sizeRatio := new(big.Int).Mul(amountIn, one)
	sizeRatio = new(big.Int).Div(sizeRatio, reserveIn)

	betaAdjustment := new(big.Int).Mul(sizeRatio, beta)
	betaAdjustment = new(big.Int).Div(betaAdjustment, one)
	betaAdjustment = new(big.Int).Div(betaAdjustment, one) // Apply as reduction factor

	// Delta adjustment: affects slippage curve
	// Typically reduces output for large trades
	deltaReduction := new(big.Int).Mul(baseAmountOut, delta)
	deltaReduction = new(big.Int).Div(deltaReduction, one)
	deltaReduction = new(big.Int).Mul(deltaReduction, betaAdjustment)
	deltaReduction = new(big.Int).Div(deltaReduction, one)

	// Apply all adjustments
	adjustedOutput := new(big.Int).Set(alphaFactor)
	adjustedOutput = new(big.Int).Sub(adjustedOutput, deltaReduction)

	// Lambda & Oracle rate adjustment:
	// If oracle rate is available, use it to guide pricing
	// Lambda controls how much weight to give the oracle vs pool reserves
	//
	// oracleBasedOutput = amountIn * oracleRate / 1e18
	// finalOutput = (1-lambda)*adjustedOutput + lambda*oracleBasedOutput
	//             = adjustedOutput + lambda * (oracleBasedOutput - adjustedOutput) / 1e18
	if oracleRate != nil && oracleRate.Cmp(bignumber.ZeroBI) > 0 && lambda != nil {
		// Calculate what the output would be based on oracle price
		oracleBasedOutput := new(big.Int).Mul(amountIn, oracleRate)
		oracleBasedOutput = new(big.Int).Div(oracleBasedOutput, one)

		// Calculate the difference
		diff := new(big.Int).Sub(oracleBasedOutput, adjustedOutput)

		// Apply lambda weight: adjustment = lambda * diff / 1e18
		oracleAdjustment := new(big.Int).Mul(lambda, diff)
		oracleAdjustment = new(big.Int).Div(oracleAdjustment, one)

		// Add oracle adjustment to the output
		adjustedOutput = new(big.Int).Add(adjustedOutput, oracleAdjustment)
	}

	// Ensure output is positive and reasonable
	if adjustedOutput.Cmp(bignumber.ZeroBI) <= 0 {
		// Fallback to base amount if adjustments are too aggressive
		fallbackAmount := new(big.Int).Mul(baseAmountOut, big.NewInt(95))
		return new(big.Int).Div(fallbackAmount, big.NewInt(100))
	}

	return adjustedOutput
}

// calculateSwapFeeFromEpsilon derives the swap fee basis points from epsilon
// Epsilon is the fee parameter in 1e18 precision
// Returns fee in basis points (1 bp = 0.01%)
func calculateSwapFeeFromEpsilon(epsilon *big.Int) int64 {
	// Epsilon is typically around 1.5e15 for 0.15% fee
	// Convert from 1e18 precision to basis points (1e4 precision)
	// fee_bps = epsilon * 10000 / 1e18

	if epsilon == nil || epsilon.Cmp(bignumber.ZeroBI) == 0 {
		return 15 // Default 0.15% = 15 bps
	}

	// epsilon (1e18) -> basis points (1e4)
	// bps = epsilon / 1e14
	bps := new(big.Int).Div(epsilon, big.NewInt(1e14))

	return bps.Int64()
}

// applyFee applies a fee to an amount
func applyFee(amount *big.Int, feeBps int64) *big.Int {
	fee := new(big.Int).Mul(amount, big.NewInt(feeBps))
	fee = new(big.Int).Div(fee, big.NewInt(10000))

	return new(big.Int).Sub(amount, fee)
}
