package stabull

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrZeroDenominator       = errors.New("zero denominator")
	ErrConvergenceFailed     = errors.New("swap convergence failed")
)

const (
	// Maximum fee (0.25 in 64x64 fixed point)
	maxFeeHex = "0x4000000000000000"
)

// calculateStabullSwap implements the Stabull curve swap calculation
// Based on CurveMath.sol from https://github.com/stabull/v1-amm/blob/dev/src/CurveMath.sol
//
// This implements the iterative convergence algorithm that:
// 1. Calculates omega (fee for old state)
// 2. Iterates up to 32 times adjusting output based on psi (fee for new state)
// 3. Uses lambda to weight the fee adjustment when omega >= psi
// 4. Validates that swap doesn't move reserves outside alpha bounds
//
// Parameters:
// - alpha: Reserve ratio bounds (e.g., 0.5 = allow reserves between 25-75% of total)
// - beta: Fee threshold multiplier (defines when fees start accruing)
// - delta: Fee rate multiplier (controls fee magnitude)
// - epsilon: Base swap fee (applied as final multiplication after convergence)
// - lambda: Fee adjustment weight when omega >= psi
//
// All calculations are done in numeraire space (18 decimals)
func calculateStabullSwap(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	alpha *big.Int,
	beta *big.Int,
	delta *big.Int,
	epsilon *big.Int,
	lambda *big.Int,
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

	one := bignumber.BONE

	// Global liquidity = sum of all balances
	oGLiq := new(big.Int).Add(reserveIn, reserveOut)

	// Initial balances
	oBals := []*big.Int{
		new(big.Int).Set(reserveIn),
		new(big.Int).Set(reserveOut),
	}

	// 50/50 weights
	weights := []*big.Int{
		new(big.Int).Div(one, big.NewInt(2)),
		new(big.Int).Div(one, big.NewInt(2)),
	}

	// Calculate omega (fee for old state)
	omega := calculateFee(oGLiq, oBals, beta, delta, weights)

	// Start with negative of input (will be adjusted in loop)
	outputAmt := new(big.Int).Neg(amountIn)

	// Initialize new balances matching viewOriginSwapData:
	// After the loop: nBals[input] = balance + amt, nBals[output] = balance - amt
	nBals := []*big.Int{
		new(big.Int).Add(oBals[0], amountIn), // nBals[input] = oBals[input] + amountIn
		new(big.Int).Sub(oBals[1], amountIn), // nBals[output] = oBals[output] - amountIn
	}

	// Contract: nGLiq_ = nGLiq_.sub(amt_) but nGLiq already includes +amt from input side
	// So: nGLiq = (oBals[0] + amt) + (oBals[1]) - amt = oBals[0] + oBals[1] = oGLiq
	nGLiq := new(big.Int).Set(oGLiq)

	// Iterative convergence
	for i := 0; i < 32; i++ {
		// Calculate psi (fee for new state)
		psi := calculateFee(nGLiq, nBals, beta, delta, weights)

		// Save previous for convergence check
		prevAmount := new(big.Int).Set(outputAmt)

		// Calculate new output amount
		if omega.Cmp(psi) < 0 {
			// outputAmt = -(amountIn + omega - psi)
			outputAmt = new(big.Int).Sub(omega, psi)
			outputAmt.Add(outputAmt, amountIn)
			outputAmt.Neg(outputAmt)
		} else {
			// outputAmt = -(amountIn + lambda * (omega - psi))
			feeDiff := new(big.Int).Sub(omega, psi)
			lambdaAdj := new(big.Int).Mul(lambda, feeDiff)
			lambdaAdj.Div(lambdaAdj, one)

			outputAmt = new(big.Int).Add(amountIn, lambdaAdj)
			outputAmt.Neg(outputAmt)
		}

		// Check convergence (1e13 precision)
		prevScaled := new(big.Int).Div(new(big.Int).Abs(prevAmount), big.NewInt(1e13))
		currScaled := new(big.Int).Div(new(big.Int).Abs(outputAmt), big.NewInt(1e13))

		if prevScaled.Cmp(currScaled) == 0 {
			// Converged! Update final state
			nGLiq = new(big.Int).Add(oGLiq, amountIn)
			nGLiq.Add(nGLiq, outputAmt)
			nBals[1] = new(big.Int).Add(oBals[1], outputAmt)

			result := new(big.Int).Abs(outputAmt)

			if result.Cmp(reserveOut) >= 0 {
				return nil, ErrInsufficientLiquidity
			}

			// enforceHalts: Check alpha bounds to prevent swaps that move reserves too far
			// This matches CurveMath.sol enforceHalts() at line 203
			// Alpha defines halt boundaries relative to ideal (weighted) balance
			// For 50/50 pool with alpha=0.5:
			//   - Ideal = 50% of total liquidity
			//   - Upper halt = ideal * (1 + alpha) = 50% * 1.5 = 75%
			//   - Lower halt = ideal * (1 - alpha) = 50% * 0.5 = 25%

			oGLiq := new(big.Int).Add(reserveIn, reserveOut)
			nGLiq := new(big.Int).Add(
				new(big.Int).Add(reserveIn, amountIn),
				new(big.Int).Sub(reserveOut, result),
			)

			// oBals and nBals for input and output tokens
			oBalsIn := reserveIn
			oBalsOut := reserveOut
			nBalsIn := new(big.Int).Add(reserveIn, amountIn)
			nBalsOut := new(big.Int).Sub(reserveOut, result)

			// Weight is 0.5 (50%) for both tokens in a 50/50 pool
			weight := new(big.Int).Div(one, big.NewInt(2)) // 0.5e18

			// Check input token halts
			if err := enforceHaltsForToken(oGLiq, nGLiq, oBalsIn, nBalsIn, weight, alpha); err != nil {
				return nil, err
			}

			// Check output token halts
			if err := enforceHaltsForToken(oGLiq, nGLiq, oBalsOut, nBalsOut, weight, alpha); err != nil {
				return nil, err
			}

			// Apply epsilon fee: result = result * (ONE - epsilon) / ONE
			// In the contract: _amt = _amt.us_mul(ONE - curve.epsilon)
			oneMinusEpsilon := new(big.Int).Sub(one, epsilon)
			result = new(big.Int).Mul(result, oneMinusEpsilon)
			result.Div(result, one)

			return result, nil
		}

		// Update state for next iteration
		nGLiq = new(big.Int).Add(oGLiq, amountIn)
		nGLiq.Add(nGLiq, outputAmt)
		nBals[1] = new(big.Int).Add(oBals[1], outputAmt)
	}

	return nil, ErrConvergenceFailed
}

// calculateFee implements the fee calculation from CurveMath.sol
// Calculates total fee (omega/psi) for a given pool state
func calculateFee(
	gLiq *big.Int,
	bals []*big.Int,
	beta *big.Int,
	delta *big.Int,
	weights []*big.Int,
) *big.Int {
	psi := bignumber.ZeroBI

	for i := 0; i < len(bals); i++ {
		// ideal = gLiq * weight[i] / 1e18
		ideal := new(big.Int).Mul(gLiq, weights[i])
		ideal.Div(ideal, bignumber.BONE)

		// Calculate micro fee for this token
		microFee := calculateMicroFee(bals[i], ideal, beta, delta)
		psi = new(big.Int).Add(psi, microFee)
	}

	return psi
}

// calculateMicroFee implements per-token fee from CurveMath.sol
func calculateMicroFee(bal *big.Int, ideal *big.Int, beta *big.Int, delta *big.Int) *big.Int {
	one := bignumber.BONE
	maxFee, _ := new(big.Int).SetString(maxFeeHex, 0)

	if bal.Cmp(ideal) < 0 {
		// Balance below ideal
		// threshold = ideal * (1 - beta) / 1e18
		betaAdj := new(big.Int).Sub(one, beta)
		threshold := new(big.Int).Mul(ideal, betaAdj)
		threshold.Div(threshold, one)

		if bal.Cmp(threshold) < 0 {
			// feeMargin = threshold - bal
			feeMargin := new(big.Int).Sub(threshold, bal)

			// fee = (feeMargin * delta) / 1e18
			fee := new(big.Int).Mul(feeMargin, delta)
			fee.Div(fee, one)

			// fee = (fee * 1e18) / ideal (fixed-point division)
			fee.Mul(fee, one)
			fee.Div(fee, ideal)

			if fee.Cmp(maxFee) > 0 {
				fee = new(big.Int).Set(maxFee)
			}

			// fee = (fee * feeMargin) / 1e18
			fee.Mul(fee, feeMargin)
			fee.Div(fee, one)
			return fee
		}
		return bignumber.ZeroBI
	}

	// Balance above ideal
	// threshold = ideal * (1 + beta) / 1e18
	betaAdj := new(big.Int).Add(one, beta)
	threshold := new(big.Int).Mul(ideal, betaAdj)
	threshold.Div(threshold, one)

	if bal.Cmp(threshold) > 0 {
		// feeMargin = bal - threshold
		feeMargin := new(big.Int).Sub(bal, threshold)

		// fee = (feeMargin * delta) / 1e18
		fee := new(big.Int).Mul(feeMargin, delta)
		fee.Div(fee, one)

		// fee = (fee * 1e18) / ideal (fixed-point division)
		fee.Mul(fee, one)
		fee.Div(fee, ideal)

		if fee.Cmp(maxFee) > 0 {
			fee = new(big.Int).Set(maxFee)
		}

		// fee = (fee * feeMargin) / 1e18
		fee.Mul(fee, feeMargin)
		fee.Div(fee, one)
		return fee
	}
	return bignumber.ZeroBI
}

// enforceHaltsForToken checks alpha bounds for a single token
// Implements the logic from CurveMath.sol enforceHalts() at line 203
// Alpha defines halt boundaries relative to ideal (weighted) balance:
//   - If balance > ideal: upper halt = ideal * (1 + alpha)
//   - If balance < ideal: lower halt = ideal * (1 - alpha)
//
// Reverts if:
//  1. Balance crosses halt boundary (was inside, now outside)
//  2. Balance is outside halt and moving further away
func enforceHaltsForToken(oGLiq, nGLiq, oBal, nBal, weight, alpha *big.Int) error {
	one := bignumber.TenPowInt(18)

	// Calculate ideal balances: ideal = liquidity * weight / 1e18
	nIdeal := new(big.Int).Mul(nGLiq, weight)
	nIdeal.Div(nIdeal, one)

	if nBal.Cmp(nIdeal) > 0 {
		// Balance above ideal - check upper halt
		// upperAlpha = 1 + alpha
		upperAlpha := new(big.Int).Add(one, alpha)

		// nHalt = nIdeal * upperAlpha / 1e18
		nHalt := new(big.Int).Mul(nIdeal, upperAlpha)
		nHalt.Div(nHalt, one)

		if nBal.Cmp(nHalt) > 0 {
			// New balance exceeds upper halt
			// Calculate old halt
			oIdeal := new(big.Int).Mul(oGLiq, weight)
			oIdeal.Div(oIdeal, one)
			oHalt := new(big.Int).Mul(oIdeal, upperAlpha)
			oHalt.Div(oHalt, one)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Cmp(oHalt) < 0 {
				return fmt.Errorf("upper halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := new(big.Int).Sub(nBal, nHalt)
			oDist := new(big.Int).Sub(oBal, oHalt)
			if nDist.Cmp(oDist) > 0 {
				return fmt.Errorf("upper halt: distance increasing")
			}
		}
	} else {
		// Balance below ideal - check lower halt
		// lowerAlpha = 1 - alpha
		lowerAlpha := new(big.Int).Sub(one, alpha)

		// nHalt = nIdeal * lowerAlpha / 1e18
		nHalt := new(big.Int).Mul(nIdeal, lowerAlpha)
		nHalt.Div(nHalt, one)

		if nBal.Cmp(nHalt) < 0 {
			// New balance below lower halt
			// Calculate old halt
			oIdeal := new(big.Int).Mul(oGLiq, weight)
			oIdeal.Div(oIdeal, one)
			oHalt := new(big.Int).Mul(oIdeal, lowerAlpha)
			oHalt.Div(oHalt, one)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Cmp(oHalt) > 0 {
				return fmt.Errorf("lower halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := new(big.Int).Sub(nHalt, nBal)
			oDist := new(big.Int).Sub(oHalt, oBal)
			if nDist.Cmp(oDist) > 0 {
				return fmt.Errorf("lower halt: distance increasing")
			}
		}
	}

	return nil
}
