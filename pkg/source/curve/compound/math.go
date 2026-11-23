package compound

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// _xp returns the balances scaled to 18 decimals: balance * rate * multiplier / BONE.
func _xp(
	balances []*uint256.Int,
	rates []*uint256.Int,
	tokenPrecisionMultipliers []*uint256.Int,
) ([]*uint256.Int, error) {
	var numTokens = len(balances)
	if numTokens != len(rates) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	xp := make([]*uint256.Int, numTokens)
	for i := range numTokens {
		var rm uint256.Int
		rm.Mul(rates[i], tokenPrecisionMultipliers[i])
		xp[i] = new(uint256.Int).Mul(balances[i], &rm)
		xp[i].Div(xp[i], Bone)
	}
	return xp, nil
}

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
	var d, prevD, dP, nA, tmp, denom uint256.Int
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

		// d = (nA*s + dP*N) * d / ((nA - 1)*d + dP*(N+1)); legacy's nA-1 is signed, so nA < 1 is not handled here
		var num uint256.Int
		num.Mul(&nA, &s)
		tmp.Mul(&dP, numTokensU256)
		num.Add(&num, &tmp)
		num.Mul(&num, &d)

		var sub uint256.Int
		sub.Sub(&nA, big256.U1)
		tmp.Mul(&sub, &d)
		denom.Mul(&dP, uint256.NewInt(uint64(numTokens+1)))
		tmp.Add(&tmp, &denom)
		d.Div(&num, &tmp)

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
// Compound's getY has no A_PRECISION factor (unlike aave).
func getY(
	APrecise *uint256.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []*uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var a = APrecise
	d, err := getD(xp, a)
	if err != nil {
		return nil, err
	}
	var c uint256.Int
	c.Set(d)
	var s uint256.Int
	var nA uint256.Int
	nA.Mul(a, numTokensU256)
	for i := range numTokens {
		var _x uint256.Int
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
		c.Mul(&c, d)
		c.Div(&c, new(uint256.Int).Mul(&_x, numTokensU256))
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Mul(&c, d)
	c.Div(&c, new(uint256.Int).Mul(&nA, numTokensU256))

	var b uint256.Int
	b.Div(d, &nA)
	b.Add(&s, &b)

	var yPrev, y, denom uint256.Int
	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		// denom = 2*y + b - d; legacy errors when it is not positive
		denom.Mul(&y, uint256.NewInt(2))
		denom.Add(&denom, &b)
		if denom.Cmp(d) <= 0 {
			return nil, ErrDenominatorZero
		}
		denom.Sub(&denom, d)
		y.Mul(&y, &y)
		y.Add(&y, &c)
		y.Div(&y, &denom)
		if withinDelta(&y, &yPrev, 1) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

// GetDyUnderlying returns the output and fee of an underlying swap of dx from tokenIndexFrom to tokenIndexTo.
func GetDyUnderlying(
	balances []*uint256.Int,
	rates []*uint256.Int,
	tokenPrecisionMultipliers []*uint256.Int,
	APrecise *uint256.Int,
	swapFee *uint256.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	xp, err := _xp(balances, rates, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x uint256.Int
	x.Mul(dx, tokenPrecisionMultipliers[tokenIndexFrom])
	x.Add(&x, xp[tokenIndexFrom])
	y, err := getY(APrecise, tokenIndexFrom, tokenIndexTo, &x, xp)
	if err != nil {
		return nil, nil, err
	}
	// legacy dy = xp[j] - y goes negative when y > xp[j]; uint256 cannot hold it
	if xp[tokenIndexTo].Cmp(y) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xp[tokenIndexTo], y)
	dy.Div(&dy, tokenPrecisionMultipliers[tokenIndexTo])

	var fee uint256.Int
	fee.Mul(swapFee, &dy)
	fee.Div(&fee, FeeDenominator)
	if dy.Cmp(&fee) < 0 {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, &fee)
	return &dy, &fee, nil
}

// GetDxUnderlying returns the input dx and fee for an exact-out underlying swap of dy.
// Legacy's signed y can be negative; that family is ErrNegativeResult (decision D on the getY path).
func GetDxUnderlying(
	balances []*uint256.Int,
	rates []*uint256.Int,
	tokenPrecisionMultipliers []*uint256.Int,
	APrecise *uint256.Int,
	swapFee *uint256.Int,
	i int,
	j int,
	dy *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	xp, err := _xp(balances, rates, tokenPrecisionMultipliers)
	if err != nil {
		return nil, nil, err
	}

	var gross uint256.Int
	gross.Mul(dy, FeeDenominator)
	var den uint256.Int
	den.Sub(FeeDenominator, swapFee)
	gross.Div(&gross, &den)
	var fee uint256.Int
	fee.Sub(&gross, dy)

	// y = xp[j] - (dy*F/(F-swap)) * mult[j]; negative when the gross out exceeds xp[j]
	var scaled uint256.Int
	scaled.Mul(&gross, tokenPrecisionMultipliers[j])
	if xp[j].Cmp(&scaled) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var y uint256.Int
	y.Sub(xp[j], &scaled)

	x, err := getY(APrecise, j, i, &y, xp)
	if err != nil {
		return nil, nil, err
	}
	// dx = (x - xp[i]) / mult[i]; negative when x < xp[i]
	if x.Cmp(xp[i]) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dx uint256.Int
	dx.Sub(x, xp[i])
	dx.Div(&dx, tokenPrecisionMultipliers[i])
	return &dx, &fee, nil
}
