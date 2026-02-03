package stabull

import (
	"errors"
	"fmt"

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

	var oGLiq, outputAmt, lambdaAdj, prevScaled, currScaled uint256.Int

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
		new(uint256.Int).Add(oBals[0], amountIn), // nBals[input] = oBals[input] + amountIn
		new(uint256.Int).Sub(oBals[1], amountIn), // nBals[output] = oBals[output] - amountIn
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
			nGLiq.Add(&oGLiq, amountIn)
			nGLiq.Add(nGLiq, &outputAmt)
			nBals[1] = new(uint256.Int).Add(oBals[1], &outputAmt)

			result := new(uint256.Int).Abs(&outputAmt)

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

			oGLiq := new(uint256.Int).Add(reserveIn, reserveOut)
			nGLiq := new(uint256.Int).Add(
				new(uint256.Int).Add(reserveIn, amountIn),
				new(uint256.Int).Sub(reserveOut, result),
			)

			// oBals and nBals for input and output tokens
			oBalsIn := reserveIn
			oBalsOut := reserveOut
			nBalsIn := new(uint256.Int).Add(reserveIn, amountIn)
			nBalsOut := new(uint256.Int).Sub(reserveOut, result)

			// Weight is 0.5 (50%) for both tokens in a 50/50 pool
			weight := new(uint256.Int).Div(big256.U2Pow64, uint256.NewInt(2)) // 0.5e18

			// Check input token halts
			if err := enforceHaltsForToken(oGLiq, nGLiq, oBalsIn, nBalsIn, weight, alpha); err != nil {
				return nil, err
			}

			// Check output token halts
			if err := enforceHaltsForToken(oGLiq, nGLiq, oBalsOut, nBalsOut, weight, alpha); err != nil {
				return nil, err
			}

			return result, nil
		}

		// Update state for next iteration
		nGLiq = new(uint256.Int).Add(&oGLiq, amountIn)
		nGLiq.Add(nGLiq, &outputAmt)
		nBals[1] = new(uint256.Int).Add(oBals[1], &outputAmt)
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
	psi := big256.U0

	for i := 0; i < len(bals); i++ {
		// ideal = gLiq * weight[i] / 1e18
		ideal := new(uint256.Int).Mul(gLiq, weights[i])
		ideal.Div(ideal, big256.U2Pow64)

		// Calculate micro fee for this token
		microFee := calculateMicroFee(bals[i], ideal, beta, delta)
		psi = new(uint256.Int).Add(psi, microFee)
	}

	return psi
}

// calculateMicroFee implements per-token fee from CurveMath.sol
func calculateMicroFee(bal *uint256.Int, ideal *uint256.Int, beta *uint256.Int, delta *uint256.Int) *uint256.Int {
	one := big256.U2Pow64

	if bal.Cmp(ideal) < 0 {
		// Balance below ideal
		// threshold = ideal * (1 - beta) / 1e18
		betaAdj := new(uint256.Int).Sub(one, beta)
		threshold := new(uint256.Int).Mul(ideal, betaAdj)
		threshold.Div(threshold, one)

		if bal.Cmp(threshold) < 0 {
			// feeMargin = threshold - bal
			feeMargin := new(uint256.Int).Sub(threshold, bal)

			// fee = (feeMargin * delta) / 1e18
			fee := new(uint256.Int).Mul(feeMargin, delta)
			fee.Div(fee, one)

			// fee = (fee * 1e18) / ideal (fixed-point division)
			fee.Mul(fee, one)
			fee.Div(fee, ideal)

			if fee.Cmp(maxFee) > 0 {
				fee = new(uint256.Int).Set(maxFee)
			}

			// fee = (fee * feeMargin) / 1e18
			fee.Mul(fee, feeMargin)
			fee.Div(fee, one)
			return fee
		}
		return big256.U0
	}

	// Balance above ideal
	// threshold = ideal * (1 + beta) / 1e18
	betaAdj := new(uint256.Int).Add(one, beta)
	threshold := new(uint256.Int).Mul(ideal, betaAdj)
	threshold.Div(threshold, one)

	if bal.Cmp(threshold) > 0 {
		// feeMargin = bal - threshold
		feeMargin := new(uint256.Int).Sub(bal, threshold)

		// fee = (feeMargin * delta) / 1e18
		fee := new(uint256.Int).Mul(feeMargin, delta)
		fee.Div(fee, one)

		// fee = (fee * 1e18) / ideal (fixed-point division)
		fee.Mul(fee, one)
		fee.Div(fee, ideal)

		if fee.Cmp(maxFee) > 0 {
			fee = new(uint256.Int).Set(maxFee)
		}

		// fee = (fee * feeMargin) / 1e18
		fee.Mul(fee, feeMargin)
		fee.Div(fee, one)
		return fee
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
	one := big256.U2Pow64

	// Calculate ideal balances: ideal = liquidity * weight / 1e18
	nIdeal := new(uint256.Int).Mul(nGLiq, weight)
	nIdeal.Div(nIdeal, one)

	if nBal.Cmp(nIdeal) > 0 {
		// Balance above ideal - check upper halt
		// upperAlpha = 1 + alpha
		upperAlpha := new(uint256.Int).Add(one, alpha)

		// nHalt = nIdeal * upperAlpha / 1e18
		nHalt := new(uint256.Int).Mul(nIdeal, upperAlpha)
		nHalt.Div(nHalt, one)

		if nBal.Cmp(nHalt) > 0 {
			// New balance exceeds upper halt
			// Calculate old halt
			oIdeal := new(uint256.Int).Mul(oGLiq, weight)
			oIdeal.Div(oIdeal, one)
			oHalt := new(uint256.Int).Mul(oIdeal, upperAlpha)
			oHalt.Div(oHalt, one)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Cmp(oHalt) < 0 {
				return fmt.Errorf("upper halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := new(uint256.Int).Sub(nBal, nHalt)
			oDist := new(uint256.Int).Sub(oBal, oHalt)
			if nDist.Cmp(oDist) > 0 {
				return fmt.Errorf("upper halt: distance increasing")
			}
		}
	} else {
		// Balance below ideal - check lower halt
		// lowerAlpha = 1 - alpha
		lowerAlpha := new(uint256.Int).Sub(one, alpha)

		// nHalt = nIdeal * lowerAlpha / 1e18
		nHalt := new(uint256.Int).Mul(nIdeal, lowerAlpha)
		nHalt.Div(nHalt, one)

		if nBal.Cmp(nHalt) < 0 {
			// New balance below lower halt
			// Calculate old halt
			oIdeal := new(uint256.Int).Mul(oGLiq, weight)
			oIdeal.Div(oIdeal, one)
			oHalt := new(uint256.Int).Mul(oIdeal, lowerAlpha)
			oHalt.Div(oHalt, one)

			// Check if we crossed the boundary (was inside, now outside)
			if oBal.Cmp(oHalt) > 0 {
				return fmt.Errorf("lower halt: crossed boundary")
			}

			// Check if distance from halt is increasing
			nDist := new(uint256.Int).Sub(nHalt, nBal)
			oDist := new(uint256.Int).Sub(oHalt, oBal)
			if nDist.Cmp(oDist) > 0 {
				return fmt.Errorf("lower halt: distance increasing")
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
