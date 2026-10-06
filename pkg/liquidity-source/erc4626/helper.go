package erc4626

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// GetClosestRate converts amount using rates sampled at PrefetchAmounts (rates[i] = preview(PrefetchAmounts[i])).
// It interpolates linearly between samples, through (0, 0) below the first one and at the last sample's rate
// beyond the last one. isExactOut: amount is an amountOut, and it returns the min amountIn converting to at least
// amount; CalcAmountIn reverses exact-in, so both directions must share this one curve.
func GetClosestRate(rates []*uint256.Int, amount *uint256.Int, isExactOut bool) (*uint256.Int, error) {
	xLo, yLo := u256.U0, u256.U0
	var sampled bool
	for i, y := range rates {
		if y == nil || i >= len(PrefetchAmounts) {
			continue
		}
		x := PrefetchAmounts[i]
		if isExactOut {
			if yLo.Lt(amount) && !y.Lt(amount) {
				return interpolateAmountIn(xLo, yLo, x, y, amount), nil
			}
		} else if !x.Lt(amount) {
			return interpolateAmountOut(xLo, yLo, x, y, amount), nil
		}
		xLo, yLo, sampled = x, y, true
	}
	if !sampled || yLo.IsZero() {
		return nil, ErrInvalidRate
	}

	// beyond the last sample, at its rate
	var result uint256.Int
	if isExactOut {
		return u256.MulDivUp(&result, amount, xLo, yLo), nil
	}
	return u256.MulDivDown(&result, amount, yLo, xLo), nil
}

// interpolateAmountOut returns floor(y) on the line through (xLo, yLo) and (xHi, yHi) at x in (xLo, xHi].
// If (xLo, yLo) lies on the floored line through (0, 0) and (xHi, yHi), as for a linear vault, it uses that
// line instead: it matches on-chain to the wei, while interpolating would carry yLo's rounding error.
func interpolateAmountOut(xLo, yLo, xHi, yHi, x *uint256.Int) *uint256.Int {
	var result, dx, dy uint256.Int
	if u256.MulDivDown(&result, xLo, yHi, xHi).Eq(yLo) {
		return u256.MulDivDown(&result, x, yHi, xHi)
	}
	dx.Sub(xHi, xLo)
	if !yHi.Lt(yLo) {
		u256.MulDivDown(&result, result.Sub(x, xLo), dy.Sub(yHi, yLo), &dx)
		return result.Add(yLo, &result)
	}
	u256.MulDivUp(&result, result.Sub(x, xLo), dy.Sub(yLo, yHi), &dx)
	return result.Sub(yLo, &result)
}

// interpolateAmountIn returns the min x with interpolateAmountOut(x) >= y, for y in (yLo, yHi].
func interpolateAmountIn(xLo, yLo, xHi, yHi, y *uint256.Int) *uint256.Int {
	var result, dx, dy uint256.Int
	if u256.MulDivDown(&result, xLo, yHi, xHi).Eq(yLo) {
		return u256.MulDivUp(&result, y, xHi, yHi)
	}
	u256.MulDivUp(&result, result.Sub(y, yLo), dx.Sub(xHi, xLo), dy.Sub(yHi, yLo))
	return result.Add(xLo, &result)
}
