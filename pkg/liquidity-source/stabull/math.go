package stabull

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrConvergenceFailed     = errors.New("swap convergence failed")

	// Maximum fee (0.25 in 64x64 fixed point)
	maxFee = uint256.MustFromHex("0x4000000000000000")
)

// calculateTrade implements the Stabull curve swap calculation
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
func calculateTrade(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	alpha *uint256.Int,
	beta *uint256.Int,
	delta *uint256.Int,
	epsilon *uint256.Int,
	lambda *uint256.Int,
) (*uint256.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	} else if reserveIn == nil || reserveIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	} else if reserveOut == nil || reserveOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	var oGLiq, outputAmt, lambdaAdj, prevScaled, currScaled, nBalIn, nBalOut uint256.Int

	// Global liquidity = sum of all balances
	oGLiq.Add(reserveIn, reserveOut)

	// Initial balances
	oBals := []*uint256.Int{reserveIn.Clone(), reserveOut.Clone()}

	// 50/50 weights
	weights := []*uint256.Int{Weight50, Weight50}

	// Calculate omega (fee for old state)
	omega := calculateFee(&oGLiq, oBals, beta, delta, weights)

	// Start with negative of input (will be adjusted in loop)
	outputAmt.Neg(amountIn)

	// Initialize new balances matching viewOriginSwapData:
	// After the loop: nBals[input] = balance + amt, nBals[output] = balance - amt
	nBals := []*uint256.Int{
		nBalIn.Add(oBals[0], amountIn),  // nBals[input] = oBals[input] + amountIn
		nBalOut.Sub(oBals[1], amountIn), // nBals[output] = oBals[output] - amountIn
	}

	// Contract: nGLiq_ = nGLiq_.sub(amt_) but nGLiq already includes +amt from input side
	// So: nGLiq = (oBals[0] + amt) + (oBals[1]) - amt = oBals[0] + oBals[1] = oGLiq
	nGLiq := oGLiq.Clone()

	// Iterative convergence
	for range 32 {
		// Calculate psi (fee for new state)
		psi := calculateFee(nGLiq, nBals, beta, delta, weights)

		// Save previous for convergence check
		prevAmount := outputAmt.Clone()

		// Calculate new output amount
		if omega.Lt(psi) {
			// outputAmt = -(amountIn + omega - psi)
			outputAmt.Sub(omega, psi)
			outputAmt.Add(&outputAmt, amountIn)
			outputAmt.Neg(&outputAmt)
		} else {
			// outputAmt = -(amountIn + lambda * (omega - psi))
			feeDiff := lambdaAdj.Sub(omega, psi)
			lambdaAdj.MulDivOverflow(lambda, feeDiff, big256.U2Pow64)

			outputAmt.Add(amountIn, &lambdaAdj)
			outputAmt.Neg(&outputAmt)
		}

		// Check convergence (1e13 precision)
		prevScaled.Div(prevScaled.Abs(prevAmount), ConvergencePrecision)
		currScaled.Div(currScaled.Abs(&outputAmt), ConvergencePrecision)

		if prevScaled.Eq(&currScaled) {
			// Converged! Update final state
			result := outputAmt.Abs(&outputAmt)
			if !result.Lt(reserveOut) {
				return nil, ErrInsufficientLiquidity
			}

			// enforceHalts: Check alpha bounds to prevent swaps that move reserves too far
			// This matches CurveMath.sol enforceHalts() at line 203
			// Alpha defines halt boundaries relative to ideal (weighted) balance
			// For 50/50 pool with alpha=0.5:
			//   - Ideal = 50% of total liquidity
			//   - Upper halt = ideal * (1 + alpha) = 50% * 1.5 = 75%
			//   - Lower halt = ideal * (1 - alpha) = 50% * 0.5 = 25%

			// oBals and nBals for input and output tokens
			nBalIn.Add(reserveIn, amountIn)
			nBalOut.Sub(reserveOut, result)
			nGLiq.Add(&nBalIn, &nBalOut)

			// Check input token halts
			if err := enforceHaltsForToken(&oGLiq, nGLiq, reserveIn, &nBalIn, weights[0], alpha); err != nil {
				return nil, err
			}

			// Check output token halts
			if err := enforceHaltsForToken(&oGLiq, nGLiq, reserveOut, &nBalOut, weights[1], alpha); err != nil {
				return nil, err
			}

			return result, nil
		}

		// Update state for next iteration
		nGLiq.Add(&oGLiq, amountIn)
		nGLiq.Add(nGLiq, &outputAmt)
		nBals[1].Add(oBals[1], &outputAmt)
	}

	return nil, ErrConvergenceFailed
}

// calculateFee implements the fee calculation from CurveMath.sol
// Calculates total fee (omega/psi) for a given pool state
func calculateFee(
	gLiq *uint256.Int,
	bals []*uint256.Int,
	beta *uint256.Int,
	delta *uint256.Int,
	weights []*uint256.Int,
) *uint256.Int {
	var psi uint256.Int

	for i, bal := range bals {
		// ideal = gLiq * weight[i] / 2^64
		ideal := usMul(gLiq.Clone(), weights[i])

		// Calculate micro fee for this token
		microFee := calculateMicroFee(bal, ideal, beta, delta)
		psi.Add(&psi, microFee)
	}

	return &psi
}

// calculateMicroFee implements per-token fee from CurveMath.sol
func calculateMicroFee(bal *uint256.Int, ideal *uint256.Int, beta *uint256.Int, delta *uint256.Int) *uint256.Int {
	if bal.Lt(ideal) {
		// Balance below ideal
		// threshold = ideal * (1 - beta) / 2^64
		betaAdj := new(uint256.Int).Sub(big256.U2Pow64, beta)
		threshold := usMul(ideal.Clone(), betaAdj)

		if bal.Lt(threshold) {
			// feeMargin = threshold - bal
			feeMargin := new(uint256.Int).Sub(threshold, bal)

			// fee = (feeMargin * delta) / 2^64
			fee := usMul(feeMargin.Clone(), delta)

			// fee = (fee * 2^64) / ideal (fixed-point division)
			fee.MulDivOverflow(fee, big256.U2Pow64, ideal)

			if fee.Gt(maxFee) {
				fee.Set(maxFee)
			}

			// fee = (fee * feeMargin) / 2^64
			return usMul(fee, feeMargin)
		}
		return big256.U0
	}

	// Balance above ideal
	// threshold = ideal * (1 + beta) / 2^64
	betaAdj := new(uint256.Int).Add(big256.U2Pow64, beta)
	threshold := usMul(ideal.Clone(), betaAdj)

	if bal.Gt(threshold) {
		// feeMargin = bal - threshold
		feeMargin := new(uint256.Int).Sub(bal, threshold)

		// fee = (feeMargin * delta) / 2^64
		fee := usMul(feeMargin.Clone(), delta)

		// fee = (fee * 2^64) / ideal (fixed-point division)
		fee.MulDivOverflow(fee, big256.U2Pow64, ideal)

		if fee.Gt(maxFee) {
			fee.Set(maxFee)
		}

		// fee = (fee * feeMargin) / 2^64
		return usMul(fee, feeMargin)
	}
	return big256.U0
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
func enforceHaltsForToken(oGLiq, nGLiq, oBal, nBal, weight, alpha *uint256.Int) error {
	// Calculate ideal balances: ideal = liquidity * weight / 2^64
	nIdeal := usMul(nGLiq.Clone(), weight)

	if nBal.Gt(nIdeal) {
		// Balance above ideal - check upper halt
		// upperAlpha = 2^64 + alpha
		upperAlpha := new(uint256.Int).Add(big256.U2Pow64, alpha)

		// nHalt = nIdeal * upperAlpha / 2^64
		nHalt := usMul(nIdeal, upperAlpha)

		if nBal.Gt(nHalt) {
			// New balance exceeds upper halt
			// Calculate old halt
			oIdeal := usMul(oGLiq.Clone(), weight)
			oHalt := usMul(oIdeal, upperAlpha)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Lt(oHalt) {
				return errors.New("upper halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := nHalt.Sub(nBal, nHalt)
			oDist := oHalt.Sub(oBal, oHalt)
			if nDist.Gt(oDist) {
				return errors.New("upper halt: distance increasing")
			}
		}
	} else {
		// Balance below ideal - check lower halt
		// lowerAlpha = 2^64 - alpha
		lowerAlpha := new(uint256.Int).Sub(big256.U2Pow64, alpha)

		// nHalt = nIdeal * lowerAlpha / 2^64
		nHalt := usMul(nIdeal, lowerAlpha)

		if nBal.Lt(nHalt) {
			// New balance below lower halt
			// Calculate old halt
			oIdeal := usMul(oGLiq.Clone(), weight)
			oHalt := usMul(oIdeal, lowerAlpha)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Gt(oHalt) {
				return errors.New("lower halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := nHalt.Sub(nHalt, nBal)
			oDist := oHalt.Sub(oHalt, oBal)
			if nDist.Gt(oDist) {
				return errors.New("lower halt: distance increasing")
			}
		}
	}

	return nil
}

func divu(x, y *uint256.Int) *uint256.Int {
	if x.Lt(big256.U2Pow192) {
		return x.Lsh(x, 64).Div(x, y)
	}

	msb := uint(x.BitLen())

	var result, hi, lo, xh, xl uint256.Int
	result.Div(xh.Lsh(x, 255-msb), xl.Rsh(xl.SubUint64(y, 1), msb-191).AddUint64(&xl, 1))

	hi.Mul(&result, hi.Rsh(y, 128))
	lo.Mul(&result, lo.And(y, big256.UMaxU128))

	xh.Rsh(x, 192)
	xl.Lsh(x, 64)

	if xl.Lt(&lo) {
		xh.SubUint64(&xh, 1)
	}
	xl.Sub(&xl, &lo) // We rely on overflow behavior here
	lo.Lsh(&hi, 128)
	if xl.Lt(&lo) {
		xh.SubUint64(&xh, 1)
	}
	xl.Sub(&xl, &lo) // We rely on overflow behavior here

	return result.Add(&result, lo.Div(&xl, y))
}

func mulu(x, y *uint256.Int) *uint256.Int {
	var lo uint256.Int
	if y.IsZero() {
		return &lo
	}

	var hi uint256.Int
	lo.Mul(x, lo.And(y, big256.UMaxU128)).Rsh(&lo, 64)
	hi.Mul(x, hi.Rsh(y, 128))

	hi.Lsh(&hi, 64)
	return lo.Add(&lo, &hi)
}

func usMul(x, y *uint256.Int) *uint256.Int {
	return x.Mul(x, y).Rsh(x, 64)
}
