package aave

import (
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var FeeDenominator = uint256.MustFromDecimal("10000000000")

const MaxLoopLimit = 256

var APrecision = uint256.NewInt(100)

// _xp returns the balances scaled by the precision multipliers.
func _xp(
	balances []*uint256.Int,
	precisionMultipliers []*uint256.Int,
) ([]*uint256.Int, error) {
	var numTokens = len(balances)
	if numTokens != len(precisionMultipliers) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	xp := make([]*uint256.Int, numTokens)
	for i := range numTokens {
		xp[i] = new(uint256.Int).Mul(balances[i], precisionMultipliers[i])
	}
	return xp, nil
}

// _getAPrecise returns the ramped A in A_PRECISION units.
func _getAPrecise(
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
) *uint256.Int {
	var t1 = futureATime
	var a1 = futureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = initialATime
		var a0 = initialA
		var span = uint256.NewInt(uint64(t1 - t0))
		var elapsed = uint256.NewInt(uint64(now - t0))
		var step uint256.Int
		if a1.Cmp(a0) > 0 {
			step.Sub(a1, a0)
			step.Mul(&step, elapsed)
			step.Div(&step, span)
			return new(uint256.Int).Add(a0, &step)
		}
		step.Sub(a0, a1)
		step.Mul(&step, elapsed)
		step.Div(&step, span)
		return new(uint256.Int).Sub(a0, &step)
	}
	return a1
}

// getD returns the StableSwap invariant D of xp. A zero sum returns zero, as the legacy simulator does.
func getD(xp []*uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	var numTokens = len(xp)
	var s uint256.Int
	for i := range numTokens {
		s.Add(&s, xp[i])
	}
	if s.IsZero() {
		return &s, nil
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var d, prevD, dP, nA, tmp, denom, mul uint256.Int
	d.Set(&s)
	nA.Mul(a, numTokensU256)
	for range MaxLoopLimit {
		dP.Set(&d)
		for j := range numTokens {
			// +1 in the denominator prevents /0 (https://github.com/curvefi/curve-contract/blob/d4e8589/contracts/pools/aave/StableSwapAave.vy#L299)
			denom.Mul(xp[j], numTokensU256)
			denom.AddUint64(&denom, 1)
			dP.Mul(&dP, &d)
			dP.Div(&dP, &denom)
		}
		prevD.Set(&d)

		// legacy rejects the product above MaxUint256 with ErrOverflow; uint256 would wrap it
		tmp.Mul(&nA, &s)
		tmp.Div(&tmp, APrecision)
		denom.Mul(&dP, numTokensU256)
		tmp.Add(&tmp, &denom)
		if _, overflow := mul.MulOverflow(&tmp, &d); overflow {
			return nil, ErrOverflow
		}

		var sub uint256.Int
		sub.Sub(&nA, APrecision)
		tmp.Mul(&sub, &d)
		tmp.Div(&tmp, APrecision)
		denom.Mul(&dP, uint256.NewInt(uint64(numTokens+1)))
		tmp.Add(&tmp, &denom)
		d.Div(&mul, &tmp)

		if withinDelta(&d, &prevD, 1) {
			return &d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

// withinDelta reports whether |x - y| <= delta.
func withinDelta(x, y *uint256.Int, delta uint64) bool {
	var diff uint256.Int
	if x.Cmp(y) > 0 {
		diff.Sub(x, y)
	} else {
		diff.Sub(y, x)
	}
	return diff.CmpUint64(delta) <= 0
}

// getY returns the balance of tokenIndexTo after the swap, given the new balance x of tokenIndexFrom.
func getY(
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []*uint256.Int,
	dCached *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var a = _getAPrecise(futureATime, futureA, initialATime, initialA)
	var d uint256.Int
	if dCached != nil {
		d.Set(dCached)
	} else {
		dd, err := getD(xp, a)
		if err != nil {
			return nil, err
		}
		d.Set(dd)
	}
	var c, s, nA, _x uint256.Int
	c.Set(&d)
	nA.Mul(a, numTokensU256)
	for i := range numTokens {
		if i == tokenIndexFrom {
			_x.Set(x)
		} else if i != tokenIndexTo {
			_x.Set(xp[i])
		} else {
			continue
		}
		if _x.IsZero() {
			return nil, ErrZero
		}
		s.Add(&s, &_x)
		c.Mul(&c, &d)
		c.Div(&c, new(uint256.Int).Mul(&_x, numTokensU256))
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Mul(&c, &d)
	c.Mul(&c, APrecision)
	c.Div(&c, new(uint256.Int).Mul(&nA, numTokensU256))
	var b uint256.Int
	b.Mul(&d, APrecision)
	b.Div(&b, &nA)
	b.Add(&s, &b)

	var yPrev, y, denom uint256.Int
	y.Set(&d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		// denom = 2*y + b - d; legacy errors when it is not positive
		denom.Mul(&y, uint256.NewInt(2))
		denom.Add(&denom, &b)
		if denom.Cmp(&d) <= 0 {
			return nil, ErrDenominatorZero
		}
		denom.Sub(&denom, &d)
		y.Mul(&y, &y)
		y.Add(&y, &c)
		y.Div(&y, &denom)
		if withinDelta(&y, &yPrev, 1) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

// _feePerToken is the per-token swap fee multiplier.
func _feePerToken(swapFee *uint256.Int, numTokens int) *uint256.Int {
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var res uint256.Int
	res.Mul(swapFee, numTokensU256)
	res.Div(&res, new(uint256.Int).Mul(new(uint256.Int).Sub(numTokensU256, big256.U1), big256.U4))
	return &res
}

// _dynamicFee returns the off-peg-adjusted fee for the pair of balances xpi and xpj.
// Legacy divides by (xpi+xpj)^2 and panics when it is zero; that failure is returned as ErrDivisionByZero.
func _dynamicFee(xpi, xpj, fee, feeMul *uint256.Int) (*uint256.Int, error) {
	if feeMul.Cmp(FeeDenominator) <= 0 {
		return fee, nil
	}
	var xps2 uint256.Int
	xps2.Add(xpi, xpj)
	xps2.Mul(&xps2, &xps2)
	if xps2.IsZero() {
		return nil, ErrDivisionByZero
	}
	var num, den uint256.Int
	num.Sub(feeMul, FeeDenominator)
	num.Mul(&num, big256.U4)
	num.Mul(&num, xpi)
	num.Mul(&num, xpj)
	num.Div(&num, &xps2)
	den.Add(&num, FeeDenominator)
	var res uint256.Int
	res.Mul(feeMul, fee)
	res.Div(&res, &den)
	return &res, nil
}
