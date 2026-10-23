package saddle

import (
	"errors"
	"math/big"
	"time"

	"github.com/holiman/uint256"
)

// maxTokens bounds the stack arrays; saddle pools have 2 to 4 tokens.
const maxTokens = 8

var (
	feeDenominator = uint256.NewInt(10_000_000_000)
	aPrecision     = uint256.NewInt(100)

	errBalancesMultipliers  = errors.New("balances must match multipliers")
	errDNotConverge         = errors.New("d does not converge")
	errSameToken            = errors.New("can't compare token to itself")
	errTokensNotInPool      = errors.New("tokens must be in pool")
	errZero                 = errors.New("zero")
	errYNotConverge         = errors.New("approximation did not converge")
	errTokenNotFound        = errors.New("token not found")
	errTokenIndexOutOfRange = errors.New("token index out of range")
	errWithdrawExceeds      = errors.New("withdraw exceeds available")
	errWithdrawMoreThanHeld = errors.New("cannot withdraw more than available")

	// Where the big.Int implementation panicked or went on with a negative intermediate value,
	// which the contracts revert on.
	errTooManyTokens  = errors.New("too many tokens")
	errOverflow       = errors.New("uint256 overflow")
	errNegative       = errors.New("negative intermediate value")
	errDivByZero      = errors.New("division by zero")
	errRampNotStarted = errors.New("amplification ramp has not started")
	errInvalidAmount  = errors.New("amount out of uint256 range")
)

/**
 * @notice Given a set of balances and precision multipliers, return the
 * precision-adjusted balances.
 *
 * @param balances an array of token balances, in their native precisions.
 * These should generally correspond with pooled tokens.
 *
 * @param precisionMultipliers an array of multipliers, corresponding to
 * the amounts in the balances array. When multiplied together they
 * should yield amounts at the pool's precision.
 *
 * @return an array of amounts "scaled" to the pool's precision (into xp)
 */
func xpMem(balances, precisionMultipliers, xp []uint256.Int) error {
	if len(balances) != len(precisionMultipliers) {
		return errBalancesMultipliers
	}
	for i := range balances {
		if _, overflow := xp[i].MulOverflow(&balances[i], &precisionMultipliers[i]); overflow {
			return errOverflow
		}
	}
	return nil
}

/**
 * @notice Calculates and returns A based on the ramp settings
 * @dev See the StableSwap paper for details
 * @return A parameter in its raw precision form
 */
func getAPrecise(futureATime int64, futureA *uint256.Int, initialATime int64, initialA *uint256.Int) (uint256.Int, error) {
	now := time.Now().Unix()
	if futureATime <= now {
		return *futureA, nil
	}
	// On chain block.timestamp.sub(initialATime) reverts before the ramp starts.
	if now < initialATime {
		return uint256.Int{}, errRampNotStarted
	}
	var a, delta uint256.Int
	elapsed, duration := uint256.NewInt(uint64(now-initialATime)), uint256.NewInt(uint64(futureATime-initialATime))
	if futureA.Gt(initialA) {
		delta.Sub(futureA, initialA)
		delta.Div(delta.Mul(&delta, elapsed), duration)
		a.Add(initialA, &delta)
	} else {
		delta.Sub(initialA, futureA)
		delta.Div(delta.Mul(&delta, elapsed), duration)
		a.Sub(initialA, &delta)
	}
	return a, nil
}

/**
 * @notice Get D, the StableSwap invariant, based on a set of balances and a particular A.
 * @param xp a precision-adjusted set of pool balances. Array should be the same cardinality
 * as the pool.
 * @param a the amplification coefficient * n * (n - 1) in A_PRECISION.
 * See the StableSwap paper for details
 * @return the invariant, at the precision of the pool (into d)
 */
func getD(xp []uint256.Int, a, d *uint256.Int) error {
	var s uint256.Int
	for i := range xp {
		if _, overflow := s.AddOverflow(&s, &xp[i]); overflow {
			return errOverflow
		}
	}
	if s.IsZero() {
		d.Clear()
		return nil
	}

	var nCoins, nPlus1, nA, nASDivAP, nASubAP, dP, prevD, num, den, tmp uint256.Int
	nCoins.SetUint64(uint64(len(xp)))
	nPlus1.SetUint64(uint64(len(xp) + 1))
	if _, overflow := nA.MulOverflow(a, &nCoins); overflow {
		return errOverflow
	}
	if _, overflow := nASDivAP.MulDivOverflow(&nA, &s, aPrecision); overflow {
		return errOverflow
	}
	if nA.Lt(aPrecision) {
		return errNegative
	}
	nASubAP.Sub(&nA, aPrecision)

	d.Set(&s)
	for range MaxLoopLimit {
		// dP = dP * d / (x * n) for every x in xp
		dP.Set(d)
		for j := range xp {
			if _, overflow := den.MulOverflow(&xp[j], &nCoins); overflow {
				return errOverflow
			} else if den.IsZero() {
				return errDivByZero
			}
			if _, overflow := dP.MulDivOverflow(&dP, d, &den); overflow {
				return errOverflow
			}
		}
		prevD.Set(d)

		// d = (nA * s / A_PRECISION + dP * n) * d / ((nA - A_PRECISION) * d / A_PRECISION + (n + 1) * dP)
		if _, overflow := num.MulOverflow(&dP, &nCoins); overflow {
			return errOverflow
		}
		if _, overflow := num.AddOverflow(&num, &nASDivAP); overflow {
			return errOverflow
		}
		if _, overflow := tmp.MulDivOverflow(&nASubAP, d, aPrecision); overflow {
			return errOverflow
		}
		if _, overflow := den.MulOverflow(&dP, &nPlus1); overflow {
			return errOverflow
		}
		if _, overflow := den.AddOverflow(&den, &tmp); overflow {
			return errOverflow
		} else if den.IsZero() {
			return errDivByZero
		}
		if _, overflow := d.MulDivOverflow(&num, d, &den); overflow {
			return errOverflow
		}

		if withinOne(d, &prevD) {
			return nil
		}
	}
	return errDNotConverge
}

/**
 * @notice Calculate the new balances of the tokens given the indexes of the token
 * that is swapped from (FROM) and the token that is swapped to (TO).
 * This function is used as a helper function to calculate how much TO token
 * the user should receive on swap.
 *
 * @param a the amplification coefficient in A_PRECISION
 * @param tokenIndexFrom index of FROM token
 * @param tokenIndexTo index of TO token
 * @param x the new total amount of FROM token
 * @param xp balances of the tokens in the pool
 * @return the amount of TO token that should remain in the pool (into y)
 */
func getY(a *uint256.Int, tokenIndexFrom, tokenIndexTo int, x *uint256.Int, xp []uint256.Int, y *uint256.Int) error {
	numTokens := len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return errSameToken
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return errTokensNotInPool
	}
	var d uint256.Int
	if err := getD(xp, a, &d); err != nil {
		return err
	}

	var nCoins, c, s, nA, xi, den uint256.Int
	nCoins.SetUint64(uint64(numTokens))
	c.Set(&d)
	if _, overflow := nA.MulOverflow(a, &nCoins); overflow {
		return errOverflow
	}
	for i := range numTokens {
		if i == tokenIndexFrom {
			xi.Set(x)
		} else if i != tokenIndexTo {
			xi.Set(&xp[i])
		} else {
			continue
		}
		if xi.IsZero() {
			return errZero
		}
		if _, overflow := s.AddOverflow(&s, &xi); overflow {
			return errOverflow
		}
		// c = c * d / (x * n)
		if _, overflow := den.MulOverflow(&xi, &nCoins); overflow {
			return errOverflow
		}
		if _, overflow := c.MulDivOverflow(&c, &d, &den); overflow {
			return errOverflow
		}
	}
	if nA.IsZero() {
		return errZero
	}
	return solveY(&c, &s, &d, &nA, &nCoins, y)
}

/**
 * @notice Calculate the price of a token in the pool with given
 * precision-adjusted balances and a particular D.
 *
 * @dev This is accomplished via solving the invariant iteratively.
 * See the StableSwap paper and Curve.fi implementation for further details.
 *
 * x_1**2 + x1 * (sum' - (A*n**n - 1) * D / (A * n**n)) = D ** (n + 1) / (n ** (2 * n) * prod' * A)
 * x_1**2 + b*x_1 = c
 * x_1 = (x_1**2 + c) / (2*x_1 + b)
 *
 * @param a the amplification coefficient * n * (n - 1). See the StableSwap paper for details.
 * @param tokenIndex Index of token we are calculating for.
 * @param xp a precision-adjusted set of pool balances. Array should be
 * the same cardinality as the pool.
 * @param d the stableswap invariant
 * @return the price of the token, in the same precision as in xp (into y)
 */
func getYD(a *uint256.Int, tokenIndex int, xp []uint256.Int, d, y *uint256.Int) error {
	numTokens := len(xp)
	if tokenIndex >= numTokens {
		return errTokenNotFound
	}
	var nCoins, c, s, nA, den uint256.Int
	nCoins.SetUint64(uint64(numTokens))
	c.Set(d)
	if _, overflow := nA.MulOverflow(a, &nCoins); overflow {
		return errOverflow
	}
	for i := range numTokens {
		if i == tokenIndex {
			continue
		}
		if _, overflow := s.AddOverflow(&s, &xp[i]); overflow {
			return errOverflow
		}
		// c = c * d / (x * n)
		if _, overflow := den.MulOverflow(&xp[i], &nCoins); overflow {
			return errOverflow
		} else if den.IsZero() {
			return errDivByZero
		}
		if _, overflow := c.MulDivOverflow(&c, d, &den); overflow {
			return errOverflow
		}
	}
	if nA.IsZero() {
		return errDivByZero
	}
	return solveY(&c, &s, d, &nA, &nCoins, y)
}

// solveY finishes getY and getYD: c = c * d * A_PRECISION / (nA * n), b = s + d * A_PRECISION / nA,
// then iterates y = (y*y + c) / (2*y + b - d) from y = d.
func solveY(c, s, d, nA, nCoins, y *uint256.Int) error {
	var den, b, yPrev, num uint256.Int
	if _, overflow := c.MulOverflow(c, aPrecision); overflow {
		return errOverflow
	}
	if _, overflow := den.MulOverflow(nA, nCoins); overflow {
		return errOverflow
	}
	if _, overflow := c.MulDivOverflow(c, d, &den); overflow {
		return errOverflow
	}
	if _, overflow := b.MulDivOverflow(d, aPrecision, nA); overflow {
		return errOverflow
	}
	if _, overflow := b.AddOverflow(&b, s); overflow {
		return errOverflow
	}

	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(y)
		den.Lsh(y, 1)
		if _, overflow := den.AddOverflow(&den, &b); overflow {
			return errOverflow
		}
		if !den.Gt(d) {
			return errNegative
		}
		den.Sub(&den, d)
		if _, overflow := num.MulOverflow(y, y); overflow {
			return errOverflow
		}
		if _, overflow := num.AddOverflow(&num, c); overflow {
			return errOverflow
		}
		y.Div(&num, &den)
		if withinOne(y, &yPrev) {
			return nil
		}
	}
	return errYNotConverge
}

/**
 * @notice Internally calculates a swap between two tokens.
 *
 * @param tokenIndexFrom the token to sell
 * @param tokenIndexTo the token to buy
 * @param dx the number of tokens to sell.
 * @return dy the number of tokens the user will get, 0 when the swap gives nothing
 * @return dyFee the associated fee, at the pool's precision
 */
func calculateSwap(
	balances, tokenPrecisionMultipliers []uint256.Int,
	a, swapFee *uint256.Int,
	tokenIndexFrom, tokenIndexTo int,
	dx, dy, dyFee *uint256.Int,
) error {
	if len(balances) > maxTokens {
		return errTooManyTokens
	}
	var xpArr [maxTokens]uint256.Int
	xp := xpArr[:len(balances)]
	if err := xpMem(balances, tokenPrecisionMultipliers, xp); err != nil {
		return err
	}
	// x = dx * multiplier[from] + xp[from]
	var x, y uint256.Int
	if _, overflow := x.MulOverflow(dx, &tokenPrecisionMultipliers[tokenIndexFrom]); overflow {
		return errOverflow
	}
	if _, overflow := x.AddOverflow(&x, &xp[tokenIndexFrom]); overflow {
		return errOverflow
	}
	if err := getY(a, tokenIndexFrom, tokenIndexTo, &x, xp, &y); err != nil {
		return err
	}
	// dy = xp[to] - y - 1, less the fee, at token to's precision
	if !xp[tokenIndexTo].Gt(&y) {
		dy.Clear()
		dyFee.Clear()
		return nil
	}
	dy.Sub(&xp[tokenIndexTo], &y)
	dy.SubUint64(dy, 1)
	if _, overflow := dyFee.MulDivOverflow(dy, swapFee, feeDenominator); overflow {
		return errOverflow
	}
	if tokenPrecisionMultipliers[tokenIndexTo].IsZero() {
		return errDivByZero
	}
	dy.Sub(dy, dyFee)
	dy.Div(dy, &tokenPrecisionMultipliers[tokenIndexTo])
	return nil
}

/**
 * @notice Calculate the dy, the amount of selected token that user receives and
 * the fee of withdrawing in one token
 * @param tokenAmount the amount to withdraw in the pool's precision
 * @param tokenIndex which token will be withdrawn
 * @return the amount of token user will receive (0 when nothing) and the associated swap fee
 */
func calculateWithdrawOneToken(
	balances, tokenPrecisionMultipliers []uint256.Int,
	a, swapFee, withdrawFee, lpSupply *uint256.Int,
	tokenIndex int,
	tokenAmount, dy, dySwapFee *uint256.Int,
) error {
	numTokens := len(balances)
	if tokenIndex >= numTokens {
		return errTokenIndexOutOfRange
	}
	if numTokens > maxTokens {
		return errTooManyTokens
	}
	var xpArr, xpReducedArr [maxTokens]uint256.Int
	xp, xpReduced := xpArr[:numTokens], xpReducedArr[:numTokens]
	if err := xpMem(balances, tokenPrecisionMultipliers, xp); err != nil {
		return err
	}
	var d0, d1, newY, yd uint256.Int
	if err := getD(xp, a, &d0); err != nil {
		return err
	}
	// d1 = d0 - tokenAmount * d0 / lpSupply
	if lpSupply.IsZero() {
		return errDivByZero
	}
	if _, overflow := d1.MulDivOverflow(tokenAmount, &d0, lpSupply); overflow {
		return errOverflow
	}
	if tokenAmount.Gt(&xp[tokenIndex]) {
		return errWithdrawExceeds
	}
	if d1.Gt(&d0) {
		return errNegative
	}
	d1.Sub(&d0, &d1)
	if err := getYD(a, tokenIndex, xp, &d1, &newY); err != nil {
		return err
	}

	// feePerToken = swapFee * n / ((n - 1) * 4)
	if numTokens < 2 || d0.IsZero() {
		return errDivByZero
	}
	var feePerToken uint256.Int
	if _, overflow := feePerToken.MulOverflow(swapFee, uint256.NewInt(uint64(numTokens))); overflow {
		return errOverflow
	}
	feePerToken.Div(&feePerToken, uint256.NewInt(uint64(numTokens-1)*4))

	var expected, norm, reduction uint256.Int
	for i := range numTokens {
		// xpReduced[i] = xp[i] - norm * feePerToken / FEE_DENOMINATOR, with norm the distance between
		// xp[i] * d1 / d0 and the balance after the withdrawal. norm of the withdrawn token is negative
		// when newY is above xp[i] * d1 / d0; big.Int division rounds that toward minus infinity.
		if _, overflow := expected.MulDivOverflow(&xp[i], &d1, &d0); overflow {
			return errOverflow
		}
		negative := false
		if i == tokenIndex {
			if negative = expected.Lt(&newY); negative {
				norm.Sub(&newY, &expected)
			} else {
				norm.Sub(&expected, &newY)
			}
		} else {
			norm.Sub(&xp[i], &expected)
		}
		if _, overflow := reduction.MulOverflow(&norm, &feePerToken); overflow {
			return errOverflow
		}
		if negative {
			reduction.AddUint64(&reduction, feeDenominator.Uint64()-1) // round up
			reduction.Div(&reduction, feeDenominator)
			if _, overflow := xpReduced[i].AddOverflow(&xp[i], &reduction); overflow {
				return errOverflow
			}
			continue
		}
		reduction.Div(&reduction, feeDenominator)
		if reduction.Gt(&xp[i]) {
			return errNegative
		}
		xpReduced[i].Sub(&xp[i], &reduction)
	}
	if err := getYD(a, tokenIndex, xpReduced, &d1, &yd); err != nil {
		return err
	}

	// dy = (xpReduced[i] - yd - 1) / multiplier[i]
	if !xpReduced[tokenIndex].Gt(&yd) {
		dy.Clear()
		dySwapFee.Clear()
		return nil
	}
	if tokenPrecisionMultipliers[tokenIndex].IsZero() {
		return errDivByZero
	}
	dy.Sub(&xpReduced[tokenIndex], &yd)
	dy.SubUint64(dy, 1)
	dy.Div(dy, &tokenPrecisionMultipliers[tokenIndex])

	// dySwapFee = (xp[i] - newY) / multiplier[i] - dy
	if newY.Gt(&xp[tokenIndex]) {
		return errNegative
	}
	dySwapFee.Sub(&xp[tokenIndex], &newY)
	dySwapFee.Div(dySwapFee, &tokenPrecisionMultipliers[tokenIndex])
	if dy.Gt(dySwapFee) {
		return errNegative
	}
	dySwapFee.Sub(dySwapFee, dy)

	// dy = dy * (FEE_DENOMINATOR - withdrawFee) / FEE_DENOMINATOR
	if withdrawFee.Gt(feeDenominator) {
		dy.Clear()
		return nil
	}
	var keep uint256.Int
	keep.Sub(feeDenominator, withdrawFee)
	if _, overflow := dy.MulDivOverflow(dy, &keep, feeDenominator); overflow {
		return errOverflow
	}
	return nil
}

/**
 * @notice A simple method to calculate prices from deposits or
 * withdrawals, excluding fees but including slippage.
 *
 * @param amounts an array of token amounts to deposit or withdrawal,
 * corresponding to pooledTokens, in each pooled token's native precision.
 * @param deposit whether this is a deposit or a withdrawal
 * @return if deposit was true, total amount of lp token that will be minted (0 when nothing) and if
 * deposit was false, total amount of lp token that will be burned
 */
func calculateTokenAmount(
	balances, tokenPrecisionMultipliers []uint256.Int,
	a, withdrawFee, lpSupply *uint256.Int,
	amounts []uint256.Int,
	deposit bool,
	out *uint256.Int,
) error {
	numTokens := len(balances)
	if numTokens > maxTokens {
		return errTooManyTokens
	}
	var xpArr, balances1Arr [maxTokens]uint256.Int
	xp, balances1 := xpArr[:numTokens], balances1Arr[:numTokens]
	if err := xpMem(balances, tokenPrecisionMultipliers, xp); err != nil {
		return err
	}
	var d0, d1 uint256.Int
	if err := getD(xp, a, &d0); err != nil {
		return err
	}
	for i := range numTokens {
		if deposit {
			if _, overflow := balances1[i].AddOverflow(&balances[i], &amounts[i]); overflow {
				return errOverflow
			}
		} else {
			if balances[i].Lt(&amounts[i]) {
				return errWithdrawMoreThanHeld
			}
			balances1[i].Sub(&balances[i], &amounts[i])
		}
	}
	if err := xpMem(balances1, tokenPrecisionMultipliers, xp); err != nil {
		return err
	}
	if err := getD(xp, a, &d1); err != nil {
		return err
	}
	if d0.IsZero() {
		return errDivByZero
	}
	if deposit {
		// (d1 - d0) * totalSupply / d0
		if !d1.Gt(&d0) {
			out.Clear()
			return nil
		}
		out.Sub(&d1, &d0)
		if _, overflow := out.MulDivOverflow(out, lpSupply, &d0); overflow {
			return errOverflow
		}
		return nil
	}
	// (d0 - d1) * totalSupply / d0 * FEE_DENOMINATOR / (FEE_DENOMINATOR - withdrawFee)
	if d1.Gt(&d0) {
		return errNegative
	}
	if !feeDenominator.Gt(withdrawFee) {
		return errDivByZero
	}
	var keep uint256.Int
	keep.Sub(feeDenominator, withdrawFee)
	out.Sub(&d0, &d1)
	if _, overflow := out.MulDivOverflow(out, lpSupply, &d0); overflow {
		return errOverflow
	}
	if _, overflow := out.MulDivOverflow(out, feeDenominator, &keep); overflow {
		return errOverflow
	}
	return nil
}

func dynamicFee(xpi, xpj, fee, feeMul, out *uint256.Int) error {
	if !feeMul.Gt(feeDenominator) {
		out.Set(fee)
		return nil
	}
	// feeMul * fee / ((feeMul - FEE_DENOMINATOR) * 4 * xpi * xpj / (xpi + xpj)**2 + FEE_DENOMINATOR)
	var xps2, num, den uint256.Int
	if _, overflow := xps2.AddOverflow(xpi, xpj); overflow {
		return errOverflow
	}
	if _, overflow := xps2.MulOverflow(&xps2, &xps2); overflow {
		return errOverflow
	} else if xps2.IsZero() {
		return errDivByZero
	}
	den.Sub(feeMul, feeDenominator)
	if _, overflow := den.MulOverflow(&den, uint256.NewInt(4)); overflow {
		return errOverflow
	}
	if _, overflow := den.MulOverflow(&den, xpi); overflow {
		return errOverflow
	}
	if _, overflow := den.MulDivOverflow(&den, xpj, &xps2); overflow {
		return errOverflow
	}
	if _, overflow := den.AddOverflow(&den, feeDenominator); overflow {
		return errOverflow
	}
	if _, overflow := num.MulOverflow(feeMul, fee); overflow {
		return errOverflow
	}
	out.Div(&num, &den)
	return nil
}

func getDyUnderlying(
	balances, tokenPrecisionMultipliers []uint256.Int,
	a, swapFee, offPegFeeMultiplier *uint256.Int,
	tokenIndexFrom, tokenIndexTo int,
	dx, dy, fee *uint256.Int,
) error {
	if len(balances) > maxTokens {
		return errTooManyTokens
	}
	var xpArr [maxTokens]uint256.Int
	xp := xpArr[:len(balances)]
	if err := xpMem(balances, tokenPrecisionMultipliers, xp); err != nil {
		return err
	}
	var x, y, xAvg, yAvg, rate uint256.Int
	if _, overflow := x.MulOverflow(dx, &tokenPrecisionMultipliers[tokenIndexFrom]); overflow {
		return errOverflow
	}
	if _, overflow := x.AddOverflow(&x, &xp[tokenIndexFrom]); overflow {
		return errOverflow
	}
	if err := getY(a, tokenIndexFrom, tokenIndexTo, &x, xp, &y); err != nil {
		return err
	}
	// dy = (xp[to] - y) / multiplier[to]
	if y.Gt(&xp[tokenIndexTo]) {
		return errNegative
	}
	if tokenPrecisionMultipliers[tokenIndexTo].IsZero() {
		return errDivByZero
	}
	dy.Sub(&xp[tokenIndexTo], &y)
	dy.Div(dy, &tokenPrecisionMultipliers[tokenIndexTo])

	if _, overflow := xAvg.AddOverflow(&xp[tokenIndexFrom], &x); overflow {
		return errOverflow
	}
	if _, overflow := yAvg.AddOverflow(&xp[tokenIndexTo], &y); overflow {
		return errOverflow
	}
	if err := dynamicFee(xAvg.Rsh(&xAvg, 1), yAvg.Rsh(&yAvg, 1), swapFee, offPegFeeMultiplier, &rate); err != nil {
		return err
	}
	if _, overflow := fee.MulDivOverflow(&rate, dy, feeDenominator); overflow {
		return errOverflow
	}
	if fee.Gt(dy) {
		return errNegative
	}
	dy.Sub(dy, fee)
	return nil
}

func withinOne(x, y *uint256.Int) bool {
	var diff uint256.Int
	if x.Gt(y) {
		diff.Sub(x, y)
	} else {
		diff.Sub(y, x)
	}
	return diff.LtUint64(2)
}

// The exported functions keep their big.Int signatures. Where the big.Int implementation returned a
// negative amount for a swap that gives nothing, they return 0.

func toU256(dst *uint256.Int, src *big.Int) error {
	if src == nil || src.Sign() < 0 || dst.SetFromBig(src) {
		return errInvalidAmount
	}
	return nil
}

func toU256s(src []*big.Int) ([]uint256.Int, error) {
	dst := make([]uint256.Int, len(src))
	for i := range src {
		if err := toU256(&dst[i], src[i]); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

type bigParams struct {
	balances, multipliers []uint256.Int
	a                     uint256.Int
}

func newBigParams(balances, tokenPrecisionMultipliers []*big.Int, futureATime int64, futureA *big.Int,
	initialATime int64, initialA *big.Int) (*bigParams, error) {
	var p bigParams
	var err error
	if p.balances, err = toU256s(balances); err != nil {
		return nil, err
	}
	if p.multipliers, err = toU256s(tokenPrecisionMultipliers); err != nil {
		return nil, err
	}
	var fA, iA uint256.Int
	if err = toU256(&fA, futureA); err != nil {
		return nil, err
	}
	if err = toU256(&iA, initialA); err != nil {
		return nil, err
	}
	p.a, err = getAPrecise(futureATime, &fA, initialATime, &iA)
	return &p, err
}

// CalculateSwap returns the output of dx of token tokenIndexFrom in token tokenIndexTo, and the fee at
// the pool's precision.
func CalculateSwap(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	p, err := newBigParams(balances, tokenPrecisionMultipliers, futureATime, futureA, initialATime, initialA)
	if err != nil {
		return nil, nil, err
	}
	var fee, amount, dy, dyFee uint256.Int
	if err := toU256(&fee, swapFee); err != nil {
		return nil, nil, err
	}
	if err := toU256(&amount, dx); err != nil {
		return nil, nil, err
	}
	if err := calculateSwap(p.balances, p.multipliers, &p.a, &fee, tokenIndexFrom, tokenIndexTo, &amount, &dy,
		&dyFee); err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), dyFee.ToBig(), nil
}

// CalculateRemoveLiquidityOneToken returns the amount of token tokenIndex that withdrawing tokenAmount
// of the LP token gives, and the swap fee.
func CalculateRemoveLiquidityOneToken(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	p, err := newBigParams(balances, tokenPrecisionMultipliers, futureATime, futureA, initialATime, initialA)
	if err != nil {
		return nil, nil, err
	}
	var fee, wFee, supply, amount, dy, dySwapFee uint256.Int
	for _, v := range []struct {
		dst *uint256.Int
		src *big.Int
	}{{&fee, swapFee}, {&wFee, withdrawFee}, {&supply, lpSupply}, {&amount, tokenAmount}} {
		if err := toU256(v.dst, v.src); err != nil {
			return nil, nil, err
		}
	}
	if err := calculateWithdrawOneToken(p.balances, p.multipliers, &p.a, &fee, &wFee, &supply, tokenIndex, &amount,
		&dy, &dySwapFee); err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), dySwapFee.ToBig(), nil
}

// CalculateAddLiquidityOneToken returns the LP tokens that depositing tokenAmount of token tokenIndex
// mints, and a zero fee.
func CalculateAddLiquidityOneToken(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	withdrawFee *big.Int,
	lpSupply *big.Int,
	tokenIndex int,
	tokenAmount *big.Int,
) (*big.Int, *big.Int, error) {
	p, err := newBigParams(balances, tokenPrecisionMultipliers, futureATime, futureA, initialATime, initialA)
	if err != nil {
		return nil, nil, err
	}
	if tokenIndex < 0 || tokenIndex >= len(p.balances) {
		return nil, nil, errTokenIndexOutOfRange
	}
	var wFee, supply, out uint256.Int
	if err := toU256(&wFee, withdrawFee); err != nil {
		return nil, nil, err
	}
	if err := toU256(&supply, lpSupply); err != nil {
		return nil, nil, err
	}
	amounts := make([]uint256.Int, len(p.balances))
	if err := toU256(&amounts[tokenIndex], tokenAmount); err != nil {
		return nil, nil, err
	}
	if err := calculateTokenAmount(p.balances, p.multipliers, &p.a, &wFee, &supply, amounts, true, &out); err != nil {
		return nil, nil, err
	}
	return out.ToBig(), new(big.Int), nil
}

// GetDyUnderlying returns the output of dx of token tokenIndexFrom in token tokenIndexTo after the
// dynamic (off-peg) fee, and that fee.
func GetDyUnderlying(
	balances []*big.Int,
	tokenPrecisionMultipliers []*big.Int,
	futureATime int64,
	futureA *big.Int,
	initialATime int64,
	initialA *big.Int,
	swapFee *big.Int,
	offPegFeeMultiplier *big.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *big.Int,
) (*big.Int, *big.Int, error) {
	p, err := newBigParams(balances, tokenPrecisionMultipliers, futureATime, futureA, initialATime, initialA)
	if err != nil {
		return nil, nil, err
	}
	var fee, feeMul, amount, dy, dyFee uint256.Int
	for _, v := range []struct {
		dst *uint256.Int
		src *big.Int
	}{{&fee, swapFee}, {&feeMul, offPegFeeMultiplier}, {&amount, dx}} {
		if err := toU256(v.dst, v.src); err != nil {
			return nil, nil, err
		}
	}
	if err := getDyUnderlying(p.balances, p.multipliers, &p.a, &fee, &feeMul, tokenIndexFrom, tokenIndexTo, &amount,
		&dy, &dyFee); err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), dyFee.ToBig(), nil
}
