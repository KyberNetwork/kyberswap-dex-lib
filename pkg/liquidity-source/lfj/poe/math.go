package poe

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Scales matching the on-chain contracts (see native.md): price is tokenY per
// tokenX scaled 1e24, alpha (concentration) is BPS (10000 = 1.0x), fee is HBPS
// (1e6 = 100%).
var (
	precision = bignumber.TenPowInt(24)
	bps       = big.NewInt(10_000)
	hbps      = big.NewInt(1_000_000)
)

// quote is the result of getQuote: the net output, the actual (possibly
// partial) input consumed, and the fee — always denominated in tokenY,
// charged off the input on Y->X and off the output on X->Y.
type quote struct {
	amountOut *big.Int
	actualIn  *big.Int
	feeIn     *big.Int
	feeOut    *big.Int
}

// ceilDiv returns ceil(a/b) into res (a, b > 0). Safe when res aliases a.
func ceilDiv(res, a, b *big.Int) *big.Int {
	res.Add(a, b)
	res.Sub(res, big.NewInt(1))
	return res.Div(res, b)
}

// feeInclusive returns ceil(amount*fee/HBPS) — fee taken out of a gross amount.
func feeInclusive(amount, fee *big.Int) *big.Int {
	num := new(big.Int).Mul(amount, fee)
	return ceilDiv(num, num, hbps)
}

// feeExclusive returns ceil(amount*fee/(HBPS-fee)) — fee added on top of a net amount.
func feeExclusive(amount, fee *big.Int) *big.Int {
	denom := new(big.Int).Sub(hbps, fee)
	num := new(big.Int).Mul(amount, fee)
	return ceilDiv(num, num, denom)
}

// liquidity mirrors the on-chain `_liquidity`: the concentrated-curve
// liquidity L and floor/ceil of sqrt(alpha*price). px_y^2 and delta can
// exceed 2^256 for extreme reserve/price combinations, hence big.Int.
func liquidity(x, y, p, alpha *big.Int) (l, sqrtAP, ceilSqrtAP *big.Int) {
	alphaMinusBps := new(big.Int).Sub(alpha, bps)

	pxTerm := new(big.Int).Mul(p, x)
	pxY := new(big.Int).Mul(y, precision)
	pxY.Add(pxY, pxTerm)

	inner := new(big.Int).Mul(y, precision)
	inner.Mul(inner, alphaMinusBps)
	inner.Lsh(inner, 2)
	inner.Div(inner, bps)
	inner.Mul(inner, pxTerm)

	delta := new(big.Int).Mul(pxY, pxY)
	delta.Add(delta, inner)

	scaled := new(big.Int).Mul(alpha, p)
	scaled.Mul(scaled, precision)
	scaled.Div(scaled, bps)

	sqrtAP = new(big.Int).Sqrt(scaled)
	ceilSqrtAP = new(big.Int).Set(sqrtAP)
	check := new(big.Int).Mul(sqrtAP, sqrtAP)
	if check.Cmp(scaled) < 0 {
		ceilSqrtAP.Add(ceilSqrtAP, big.NewInt(1))
	}

	sqrtDelta := new(big.Int).Sqrt(delta)
	l = pxY.Add(pxY, sqrtDelta)
	l.Mul(l, alpha)
	denom := new(big.Int).Mul(ceilSqrtAP, alphaMinusBps)
	denom.Lsh(denom, 1)
	l.Div(l, denom)

	return l, sqrtAP, ceilSqrtAP
}

// virtualReserves mirrors the on-chain `_virtual_reserves`: returns the
// virtual reserve of the input token, the virtual reserve of the output
// token, and the real reserve of the output token (rounding is
// direction-dependent to match the on-chain contract bit-for-bit).
func virtualReserves(x, y, p, alpha *big.Int, xToY bool) (vin, vout, rout *big.Int) {
	l, sqrtAP, ceilSqrtAP := liquidity(x, y, p, alpha)

	if xToY {
		num := new(big.Int).Mul(l, precision)
		vin = ceilDiv(num, num, sqrtAP)
		vin.Add(vin, x)

		vout = new(big.Int).Mul(l, sqrtAP)
		vout.Mul(vout, bps)
		denom := new(big.Int).Mul(alpha, precision)
		vout.Div(vout, denom)
		vout.Add(vout, y)

		return vin, vout, new(big.Int).Set(y)
	}

	num := new(big.Int).Mul(l, ceilSqrtAP)
	num.Mul(num, bps)
	denom := new(big.Int).Mul(alpha, precision)
	vin = ceilDiv(num, num, denom)
	vin.Add(vin, y)

	vout = new(big.Int).Mul(l, precision)
	vout.Div(vout, ceilSqrtAP)
	vout.Add(vout, x)

	return vin, vout, new(big.Int).Set(x)
}

func bigMin(a, b *big.Int) *big.Int {
	if a.Cmp(b) <= 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

// stablePhase mirrors the on-chain `_stable`: trades the pool toward 50/50
// value at the flat oracle price before handing the remainder to the
// concentrated curve.
func stablePhase(reserveX, reserveY, amount, price, alpha *big.Int, xToY bool) (
	vin, vout, rout, stableIn, stableOut *big.Int,
) {
	rx, ry := new(big.Int).Set(reserveX), new(big.Int).Set(reserveY)
	stableIn, stableOut = new(big.Int), new(big.Int)

	pRx := new(big.Int).Mul(price, rx)
	ryPrecision := new(big.Int).Mul(ry, precision)

	if xToY && pRx.Cmp(ryPrecision) < 0 {
		target := new(big.Int).Sub(ryPrecision, pRx)
		target.Div(target, new(big.Int).Lsh(price, 1))
		stableIn = bigMin(amount, target)
		stableOut = new(big.Int).Mul(stableIn, price)
		stableOut.Div(stableOut, precision)
		rx.Add(rx, stableIn)
		ry.Sub(ry, stableOut)
	} else if !xToY && pRx.Cmp(ryPrecision) > 0 {
		target := new(big.Int).Sub(pRx, ryPrecision)
		target.Div(target, new(big.Int).Lsh(precision, 1))
		stableIn = bigMin(amount, target)
		stableOut = new(big.Int).Mul(stableIn, precision)
		stableOut.Div(stableOut, price)
		rx.Sub(rx, stableOut)
		ry.Add(ry, stableIn)
	}

	vin, vout, rout = virtualReserves(rx, ry, price, alpha, xToY)
	return vin, vout, rout, stableIn, stableOut
}

// getQuote mirrors the on-chain `getQuote` (verified bit-exact against it per
// native.md): stable-phase + concentrated-curve pricing, with the input
// capped (partial fill) when the curve would pay out more than the pool
// holds, and fees always denominated in tokenY.
func getQuote(reserveX, reserveY, amountIn *big.Int, xToY bool, price, fee, alpha *big.Int) (*quote, error) {
	if amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	actualIn := new(big.Int).Set(amountIn)
	feeIn, feeOut := new(big.Int), new(big.Int)

	netAmountIn := new(big.Int).Set(amountIn)
	if !xToY {
		feeIn = feeInclusive(amountIn, fee)
		netAmountIn.Sub(amountIn, feeIn)
		if netAmountIn.Sign() <= 0 {
			return nil, ErrInvalidAmountIn
		}
	}

	vin, vout, rout, stableIn, stableOut := stablePhase(reserveX, reserveY, netAmountIn, price, alpha, xToY)

	remaining := new(big.Int).Sub(netAmountIn, stableIn)

	denom := new(big.Int).Add(vin, remaining)
	if denom.Sign() == 0 {
		return nil, ErrInvalidAmountOut
	}
	curveOut := new(big.Int).Mul(remaining, vout)
	curveOut.Div(curveOut, denom)

	var amountOut *big.Int
	if curveOut.Cmp(rout) > 0 {
		if rout.Sign() == 0 {
			return nil, ErrInsufficientLiquidity
		}

		voutMinusRout := new(big.Int).Sub(vout, rout)
		if voutMinusRout.Sign() <= 0 {
			return nil, ErrInsufficientLiquidity
		}

		amountOut = new(big.Int).Add(stableOut, rout)

		num := new(big.Int).Mul(vin, rout)
		extraIn := ceilDiv(num, num, voutMinusRout)
		actualIn = new(big.Int).Add(stableIn, extraIn)

		if xToY {
			feeOut = feeInclusive(amountOut, fee)
			amountOut.Sub(amountOut, feeOut)
		} else {
			feeIn = feeExclusive(actualIn, fee)
			actualIn.Add(actualIn, feeIn)
		}
	} else {
		amountOut = new(big.Int).Add(stableOut, curveOut)
		if xToY {
			feeOut = feeInclusive(amountOut, fee)
			amountOut.Sub(amountOut, feeOut)
		}
	}

	if amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	return &quote{amountOut: amountOut, actualIn: actualIn, feeIn: feeIn, feeOut: feeOut}, nil
}
