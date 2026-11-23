package aave

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// getYD returns the balance of tokenIndex that keeps D at d, given the other balances xp.
func getYD(
	a *uint256.Int,
	tokenIndex int,
	xp []*uint256.Int,
	d *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, ErrTokenNotFound
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var c, s, nA uint256.Int
	c.Set(d)
	nA.Mul(a, numTokensU256)
	for i := range numTokens {
		if i != tokenIndex {
			s.Add(&s, xp[i])
			c.Mul(&c, d)
			c.Div(&c, new(uint256.Int).Mul(xp[i], numTokensU256))
		}
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Mul(&c, d)
	c.Mul(&c, APrecision)
	c.Div(&c, new(uint256.Int).Mul(&nA, numTokensU256))
	var b uint256.Int
	b.Mul(d, APrecision)
	b.Div(&b, &nA)
	b.Add(&s, &b)

	var yPrev, y, denom uint256.Int
	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
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

// GetDyUnderlying quotes an underlying-token swap. Legacy returns a negative dy here as a value, so it is ErrNegativeResult.
func GetDyUnderlying(
	balances []*uint256.Int,
	precisionMultipliers []*uint256.Int,
	futureATime int64,
	futureA *uint256.Int,
	initialATime int64,
	initialA *uint256.Int,
	swapFee *uint256.Int,
	offPegFeeMultiplier *uint256.Int,
	tokenIndexFrom int,
	tokenIndexTo int,
	dx *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	xp, err := _xp(balances, precisionMultipliers)
	if err != nil {
		return nil, nil, err
	}
	var x uint256.Int
	x.Mul(dx, precisionMultipliers[tokenIndexFrom])
	x.Add(&x, xp[tokenIndexFrom])
	y, err := getY(futureATime, futureA, initialATime, initialA, tokenIndexFrom, tokenIndexTo, &x, xp, nil)
	if err != nil {
		return nil, nil, err
	}
	if xp[tokenIndexTo].Cmp(y) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xp[tokenIndexTo], y)
	dy.Div(&dy, precisionMultipliers[tokenIndexTo])

	var xi, xj uint256.Int
	xi.Add(xp[tokenIndexFrom], &x)
	xi.Div(&xi, big256.U2)
	xj.Add(xp[tokenIndexTo], y)
	xj.Div(&xj, big256.U2)
	dynamicFee, err := _dynamicFee(&xi, &xj, swapFee, offPegFeeMultiplier)
	if err != nil {
		return nil, nil, err
	}
	var fee uint256.Int
	fee.Mul(dynamicFee, &dy)
	fee.Div(&fee, FeeDenominator)
	if dy.Cmp(&fee) < 0 {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, &fee)
	return &dy, &fee, nil
}
