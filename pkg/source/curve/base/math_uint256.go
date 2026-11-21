package base

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// uint256 port of math.go. Intermediate overflow returns an error because
// the on-chain Vyper code reverts; big.Int never overflows.
var (
	errU256Overflow  = errors.New("uint256 overflow")
	errU256Underflow = errors.New("uint256 underflow")

	u256One            = uint256.NewInt(1)
	u256Two            = uint256.NewInt(2)
	u256Four           = uint256.NewInt(4)
	u256FeeDenominator = big256.FromBig(FeeDenominator)
	u256Precision      = big256.FromBig(Precision)
)

func mulU256(z, x, y *uint256.Int) error {
	if _, of := z.MulOverflow(x, y); of {
		return errU256Overflow
	}
	return nil
}

func addU256(z, x, y *uint256.Int) error {
	if _, of := z.AddOverflow(x, y); of {
		return errU256Overflow
	}
	return nil
}

func subU256(z, x, y *uint256.Int) error {
	if _, of := z.SubOverflow(x, y); of {
		return errU256Underflow
	}
	return nil
}

// _AU256 mirrors _A with now passed in, so tests can pin the clock.
func (t *PoolSimulator) _AU256(now int64) (*uint256.Int, error) {
	var a1 = big256.FromBig(t.FutureA)
	if t.FutureATime <= now {
		return a1, nil
	}
	var a0 = big256.FromBig(t.InitialA)
	if now < t.InitialATime {
		return nil, errU256Underflow
	}
	var elapsed = uint256.NewInt(uint64(now - t.InitialATime))
	var duration = uint256.NewInt(uint64(t.FutureATime - t.InitialATime))
	var rising = a1.Gt(a0)

	var delta, step, res uint256.Int
	if rising {
		delta.Sub(a1, a0)
	} else {
		delta.Sub(a0, a1)
	}
	if err := mulU256(&step, &delta, elapsed); err != nil {
		return nil, err
	}
	step.Div(&step, duration)
	if rising {
		if err := addU256(&res, a0, &step); err != nil {
			return nil, err
		}
	} else if err := subU256(&res, a0, &step); err != nil {
		return nil, err
	}
	return &res, nil
}

// getDU256 mirrors getD. Ann and S/A_PRECISION terms are computed once outside the loop.
func (t *PoolSimulator) getDU256(xp []*uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	var numTokens = len(xp)
	var s uint256.Int
	for i := range numTokens {
		if err := addU256(&s, &s, xp[i]); err != nil {
			return nil, err
		}
	}
	if s.IsZero() {
		return &s, nil
	}

	var n = big256.FromBig(t.numTokensBI)
	var nPlus1 = uint256.NewInt(uint64(numTokens + 1))
	var aPrecision = big256.FromBig(t.APrecision)
	if aPrecision.IsZero() {
		return nil, ErrZero
	}

	var nA, nAMulSDivAPrec, nASubAPrec uint256.Int
	if err := mulU256(&nA, a, n); err != nil {
		return nil, err
	}
	if err := mulU256(&nAMulSDivAPrec, &nA, &s); err != nil {
		return nil, err
	}
	nAMulSDivAPrec.Div(&nAMulSDivAPrec, aPrecision)
	if err := subU256(&nASubAPrec, &nA, aPrecision); err != nil {
		return nil, err
	}

	var d, dP, prevD, nominator, denominator uint256.Int
	var tmp2, tmp4, tmp5, tmp6, tmp7, diff uint256.Int
	d.Set(&s)
	for range MaxLoopLimit {
		dP.Set(&d)
		for j := range numTokens {
			// D_P*D overflowing uint256 reverts on-chain; big.Int reports the same as ErrDDoesNotConverge.
			if _, of := nominator.MulOverflow(&dP, &d); of {
				return nil, ErrDDoesNotConverge
			}
			if err := mulU256(&denominator, xp[j], n); err != nil {
				return nil, err
			}
			if err := addU256(&denominator, &denominator, u256One); err != nil {
				return nil, err
			}
			dP.Div(&nominator, &denominator)
		}
		prevD.Set(&d)

		if err := mulU256(&tmp5, &dP, n); err != nil {
			return nil, err
		}
		if err := addU256(&tmp6, &nAMulSDivAPrec, &tmp5); err != nil {
			return nil, err
		}
		if err := mulU256(&tmp2, &tmp6, &d); err != nil {
			return nil, err
		}

		if err := mulU256(&tmp7, &nASubAPrec, &d); err != nil {
			return nil, err
		}
		tmp4.Div(&tmp7, aPrecision)
		if err := mulU256(&tmp5, &dP, nPlus1); err != nil {
			return nil, err
		}
		if err := addU256(&tmp4, &tmp4, &tmp5); err != nil {
			return nil, err
		}
		if tmp4.IsZero() {
			return nil, ErrDenominatorZero
		}
		d.Div(&tmp2, &tmp4)

		if d.Gt(&prevD) {
			diff.Sub(&d, &prevD)
		} else {
			diff.Sub(&prevD, &d)
		}
		if !diff.Gt(u256One) {
			return &d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

// getYDU256 mirrors getYD. c and b keep the on-chain sequential floor order.
func (t *PoolSimulator) getYDU256(a *uint256.Int, tokenIndex int, xp []*uint256.Int, d *uint256.Int) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, ErrTokenNotFound
	}
	var n = big256.FromBig(t.numTokensBI)
	var aPrecision = big256.FromBig(t.APrecision)

	var c, s, tmp uint256.Int
	c.Set(d)
	for i := range numTokens {
		if i == tokenIndex {
			continue
		}
		if err := addU256(&s, &s, xp[i]); err != nil {
			return nil, err
		}
		if err := mulU256(&tmp, &c, d); err != nil {
			return nil, err
		}
		var denom uint256.Int
		if err := mulU256(&denom, xp[i], n); err != nil {
			return nil, err
		}
		if denom.IsZero() {
			return nil, ErrZero
		}
		c.Div(&tmp, &denom)
	}

	var nA uint256.Int
	if err := mulU256(&nA, a, n); err != nil {
		return nil, err
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	if err := mulU256(&tmp, &c, d); err != nil {
		return nil, err
	}
	if err := mulU256(&tmp, &tmp, aPrecision); err != nil {
		return nil, err
	}
	var denom uint256.Int
	if err := mulU256(&denom, &nA, n); err != nil {
		return nil, err
	}
	c.Div(&tmp, &denom)

	var b, dA uint256.Int
	if err := mulU256(&dA, d, aPrecision); err != nil {
		return nil, err
	}
	dA.Div(&dA, &nA)
	if err := addU256(&b, &s, &dA); err != nil {
		return nil, err
	}

	var y, yPrev, denomY, num uint256.Int
	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		if err := mulU256(&denomY, &y, u256Two); err != nil {
			return nil, err
		}
		if err := addU256(&denomY, &denomY, &b); err != nil {
			return nil, err
		}
		if !denomY.Gt(d) {
			return nil, ErrDenominatorZero
		}
		denomY.Sub(&denomY, d)
		if err := mulU256(&num, &y, &y); err != nil {
			return nil, err
		}
		if err := addU256(&num, &num, &c); err != nil {
			return nil, err
		}
		y.Div(&num, &denomY)

		var diff uint256.Int
		if y.Gt(&yPrev) {
			diff.Sub(&y, &yPrev)
		} else {
			diff.Sub(&yPrev, &y)
		}
		if !diff.Gt(u256One) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

// getYU256 mirrors GetY. It takes a as a parameter and uses x for tokenIndexFrom.
func (t *PoolSimulator) getYU256(
	tokenIndexFrom, tokenIndexTo int,
	x *uint256.Int,
	xp []*uint256.Int,
	a, d *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var n = big256.FromBig(t.numTokensBI)
	var aPrecision = big256.FromBig(t.APrecision)

	var c, s, tmp, denom uint256.Int
	c.Set(d)
	for i := range numTokens {
		var v *uint256.Int
		if i == tokenIndexFrom {
			v = x
		} else if i != tokenIndexTo {
			v = xp[i]
		} else {
			continue
		}
		if v.IsZero() {
			return nil, ErrZero
		}
		if err := addU256(&s, &s, v); err != nil {
			return nil, err
		}
		if err := mulU256(&tmp, &c, d); err != nil {
			return nil, err
		}
		if err := mulU256(&denom, v, n); err != nil {
			return nil, err
		}
		c.Div(&tmp, &denom)
	}

	var nA uint256.Int
	if err := mulU256(&nA, a, n); err != nil {
		return nil, err
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	if err := mulU256(&tmp, &c, d); err != nil {
		return nil, err
	}
	if err := mulU256(&tmp, &tmp, aPrecision); err != nil {
		return nil, err
	}
	if err := mulU256(&denom, &nA, n); err != nil {
		return nil, err
	}
	c.Div(&tmp, &denom)

	var b, dA uint256.Int
	if err := mulU256(&dA, d, aPrecision); err != nil {
		return nil, err
	}
	dA.Div(&dA, &nA)
	if err := addU256(&b, &s, &dA); err != nil {
		return nil, err
	}

	var y, yPrev, denomY, num uint256.Int
	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		if err := mulU256(&denomY, &y, u256Two); err != nil {
			return nil, err
		}
		if err := addU256(&denomY, &denomY, &b); err != nil {
			return nil, err
		}
		if !denomY.Gt(d) {
			return nil, ErrDenominatorZero
		}
		denomY.Sub(&denomY, d)
		if err := mulU256(&num, &y, &y); err != nil {
			return nil, err
		}
		if err := addU256(&num, &num, &c); err != nil {
			return nil, err
		}
		y.Div(&num, &denomY)

		var diff uint256.Int
		if y.Gt(&yPrev) {
			diff.Sub(&y, &yPrev)
		} else {
			diff.Sub(&yPrev, &y)
		}
		if !diff.Gt(u256One) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

// getDyU256 mirrors GetDy. xp is recomputed from rates and reserves, as in math.go.
func (t *PoolSimulator) getDyU256(i, j int, dx *uint256.Int, xp []*uint256.Int, a, d *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	var rateI = big256.FromBig(t.Rates[i])
	var rateJ = big256.FromBig(t.Rates[j])
	var precision = big256.FromBig(Precision)
	var feeDen = big256.FromBig(FeeDenominator)
	var swapFee = big256.FromBig(t.Info.SwapFee)

	var x, tmp uint256.Int
	if err := mulU256(&tmp, dx, rateI); err != nil {
		return nil, nil, err
	}
	tmp.Div(&tmp, precision)
	if err := addU256(&x, xp[i], &tmp); err != nil {
		return nil, nil, err
	}

	var y, err = t.getYU256(i, j, &x, xp, a, d)
	if err != nil {
		return nil, nil, err
	}

	var dy, fee uint256.Int
	if err := subU256(&dy, xp[j], y); err != nil {
		return nil, nil, err
	}
	if err := subU256(&dy, &dy, u256One); err != nil {
		return nil, nil, err
	}

	if err := mulU256(&fee, swapFee, &dy); err != nil {
		return nil, nil, err
	}
	fee.Div(&fee, feeDen)

	if err := subU256(&dy, &dy, &fee); err != nil {
		return nil, nil, err
	}
	if err := mulU256(&dy, &dy, precision); err != nil {
		return nil, nil, err
	}
	dy.Div(&dy, rateJ)

	if err := mulU256(&fee, &fee, precision); err != nil {
		return nil, nil, err
	}
	fee.Div(&fee, rateJ)

	return &dy, &fee, nil
}

// calculateWithdrawOneCoinU256 mirrors CalculateWithdrawOneCoin. xp and the withdrawn-token index come in
// from the caller, and it returns (dy, dyFee) as big.Int math does.
func (t *PoolSimulator) calculateWithdrawOneCoinU256(
	tokenAmount *uint256.Int,
	i int,
	xp []*uint256.Int,
	amp, d0, totalSupply *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	var tmp, d1 uint256.Int
	if err := mulU256(&tmp, tokenAmount, d0); err != nil {
		return nil, nil, err
	}
	tmp.Div(&tmp, totalSupply)
	if err := subU256(&d1, d0, &tmp); err != nil {
		return nil, nil, err
	}
	newY, err := t.getYDU256(amp, i, xp, &d1)
	if err != nil {
		return nil, nil, err
	}

	var nCoins = len(xp)
	var nCoinBI = uint256.NewInt(uint64(nCoins))
	var fee, feeDen uint256.Int
	if err := mulU256(&fee, big256.FromBig(t.Info.SwapFee), nCoinBI); err != nil {
		return nil, nil, err
	}
	if err := subU256(&feeDen, nCoinBI, u256One); err != nil {
		return nil, nil, err
	}
	if err := mulU256(&feeDen, u256Four, &feeDen); err != nil {
		return nil, nil, err
	}
	fee.Div(&fee, &feeDen)

	var xpReduced = make([]*uint256.Int, nCoins)
	var dxExpected, a, b uint256.Int
	for j := range nCoins {
		if err := mulU256(&a, xp[j], &d1); err != nil {
			return nil, nil, err
		}
		a.Div(&a, d0)
		if j == i {
			if err := subU256(&dxExpected, &a, newY); err != nil {
				return nil, nil, err
			}
		} else {
			if err := subU256(&dxExpected, xp[j], &a); err != nil {
				return nil, nil, err
			}
		}
		if err := mulU256(&b, &fee, &dxExpected); err != nil {
			return nil, nil, err
		}
		b.Div(&b, u256FeeDenominator)
		xpReduced[j] = new(uint256.Int)
		if err := subU256(xpReduced[j], xp[j], &b); err != nil {
			return nil, nil, err
		}
	}
	newYD, err := t.getYDU256(amp, i, xpReduced, &d1)
	if err != nil {
		return nil, nil, err
	}
	var dy, dy0 uint256.Int
	if err := subU256(&dy, xpReduced[i], newYD); err != nil {
		return nil, nil, err
	}
	if err := subU256(&dy, &dy, u256One); err != nil {
		return nil, nil, err
	}
	dy.Div(&dy, big256.FromBig(t.Multipliers[i]))
	if err := subU256(&dy0, xp[i], newY); err != nil {
		return nil, nil, err
	}
	dy0.Div(&dy0, big256.FromBig(t.Multipliers[i]))
	var dyFee uint256.Int
	if err := subU256(&dyFee, &dy0, &dy); err != nil {
		return nil, nil, err
	}
	return &dy, &dyFee, nil
}

// xpMemU256 mirrors _xpMem: xp[i] = rates[i] * balances[i] / PRECISION, floored per token.
func xpMemU256(balances, rates []*uint256.Int, precision *uint256.Int) ([]*uint256.Int, error) {
	var xp = make([]*uint256.Int, len(balances))
	for i := range balances {
		var v uint256.Int
		if err := mulU256(&v, rates[i], balances[i]); err != nil {
			return nil, err
		}
		v.Div(&v, precision)
		xp[i] = &v
	}
	return xp, nil
}

// calculateTokenAmountU256 mirrors CalculateTokenAmount. Balances and xp are passed in as
// uint256 so the caller controls the rescale.
func (t *PoolSimulator) calculateTokenAmountU256(
	amounts []*uint256.Int,
	deposit bool,
	reserves, rates []*uint256.Int,
	a, totalSupply *uint256.Int,
) (*uint256.Int, error) {
	var precision = big256.FromBig(Precision)
	var numTokens = len(reserves)
	var xp0, err = xpMemU256(reserves, rates, precision)
	if err != nil {
		return nil, err
	}
	d0, err := t.getDU256(xp0, a)
	if err != nil {
		return nil, err
	}

	var balances1 = make([]*uint256.Int, numTokens)
	for i := range numTokens {
		var v uint256.Int
		if deposit {
			if err := addU256(&v, reserves[i], amounts[i]); err != nil {
				return nil, err
			}
		} else {
			if reserves[i].Lt(amounts[i]) {
				return nil, ErrWithdrawMoreThanAvailable
			}
			if err := subU256(&v, reserves[i], amounts[i]); err != nil {
				return nil, err
			}
		}
		balances1[i] = &v
	}
	xp1, err := xpMemU256(balances1, rates, precision)
	if err != nil {
		return nil, err
	}
	d1, err := t.getDU256(xp1, a)
	if err != nil {
		return nil, err
	}

	var diff uint256.Int
	if deposit {
		if err := subU256(&diff, d1, d0); err != nil {
			return nil, err
		}
	} else {
		if err := subU256(&diff, d0, d1); err != nil {
			return nil, err
		}
	}
	if err := mulU256(&diff, &diff, totalSupply); err != nil {
		return nil, err
	}
	diff.Div(&diff, d0)
	return &diff, nil
}

// addLiquidityU256 mirrors AddLiquidity without writing to t. It returns the mint amount and
// the reserves and LP supply the reference would store, so the caller decides whether to commit.
func (t *PoolSimulator) addLiquidityU256(
	amounts, oldBalances []*uint256.Int,
	a, lpSupply, swapFee, adminFee *uint256.Int,
) (*uint256.Int, []*uint256.Int, error) {
	var nCoins = len(amounts)
	var newBalances = make([]*uint256.Int, nCoins)
	for i := range nCoins {
		var v uint256.Int
		if err := addU256(&v, oldBalances[i], amounts[i]); err != nil {
			return nil, nil, err
		}
		newBalances[i] = &v
	}
	d0, err := t.getDU256(oldBalances, a)
	if err != nil {
		return nil, nil, err
	}
	d1, err := t.getDU256(newBalances, a)
	if err != nil {
		return nil, nil, err
	}
	if !d1.Gt(d0) {
		return nil, nil, ErrD1LowerThanD0
	}
	var reserves = make([]*uint256.Int, nCoins)
	if lpSupply.IsZero() {
		copy(reserves, newBalances)
		return d1, reserves, nil
	}

	var fee, feeDen uint256.Int
	if err := mulU256(&fee, swapFee, uint256.NewInt(uint64(nCoins))); err != nil {
		return nil, nil, err
	}
	if err := mulU256(&feeDen, u256Four, uint256.NewInt(uint64(nCoins-1))); err != nil {
		return nil, nil, err
	}
	var _fee uint256.Int
	_fee.Div(&fee, &feeDen)

	var ideal, difference, tmp uint256.Int
	for i := range nCoins {
		if err := mulU256(&ideal, d1, oldBalances[i]); err != nil {
			return nil, nil, err
		}
		ideal.Div(&ideal, d0)
		if ideal.Gt(newBalances[i]) {
			if err := subU256(&difference, &ideal, newBalances[i]); err != nil {
				return nil, nil, err
			}
		} else {
			if err := subU256(&difference, newBalances[i], &ideal); err != nil {
				return nil, nil, err
			}
		}
		var feeI uint256.Int
		if err := mulU256(&feeI, &_fee, &difference); err != nil {
			return nil, nil, err
		}
		feeI.Div(&feeI, u256FeeDenominator)

		if err := mulU256(&tmp, &feeI, adminFee); err != nil {
			return nil, nil, err
		}
		tmp.Div(&tmp, u256FeeDenominator)
		var reserve uint256.Int
		if err := subU256(&reserve, newBalances[i], &tmp); err != nil {
			return nil, nil, err
		}
		reserves[i] = &reserve

		var nb uint256.Int
		if err := subU256(&nb, newBalances[i], &feeI); err != nil {
			return nil, nil, err
		}
		newBalances[i] = &nb
	}
	d2, err := t.getDU256(newBalances, a)
	if err != nil {
		return nil, nil, err
	}
	var mint uint256.Int
	if err := subU256(&mint, d2, d0); err != nil {
		return nil, nil, err
	}
	if err := mulU256(&mint, lpSupply, &mint); err != nil {
		return nil, nil, err
	}
	mint.Div(&mint, d0)
	return &mint, reserves, nil
}

// removeLiquidityOneCoinU256 mirrors RemoveLiquidityOneCoin without writing to t. It returns dy and
// the reserve and LP supply the reference would store.
func (t *PoolSimulator) removeLiquidityOneCoinU256(
	tokenAmount *uint256.Int,
	i int,
	xp []*uint256.Int,
	reserves []*uint256.Int,
	amp, d0, totalSupply, adminFee *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	dy, dyFee, err := t.calculateWithdrawOneCoinU256(tokenAmount, i, xp, amp, d0, totalSupply)
	if err != nil {
		return nil, nil, nil, err
	}
	var tmp, fee uint256.Int
	if err := mulU256(&tmp, dyFee, adminFee); err != nil {
		return nil, nil, nil, err
	}
	tmp.Div(&tmp, u256FeeDenominator)
	if err := addU256(&fee, dy, &tmp); err != nil {
		return nil, nil, nil, err
	}
	var reserve uint256.Int
	if err := subU256(&reserve, reserves[i], &fee); err != nil {
		return nil, nil, nil, err
	}
	var lpSupply uint256.Int
	if err := subU256(&lpSupply, totalSupply, tokenAmount); err != nil {
		return nil, nil, nil, err
	}
	return dy, &reserve, &lpSupply, nil
}

// getVirtualPriceU256 mirrors GetVirtualPrice: D*PRECISION / LpSupply, and D.
func (t *PoolSimulator) getVirtualPriceU256(d, lpSupply *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if lpSupply.IsZero() {
		return nil, nil, ErrDenominatorZero
	}
	var v uint256.Int
	if err := mulU256(&v, d, u256Precision); err != nil {
		return nil, nil, err
	}
	v.Div(&v, lpSupply)
	return &v, d, nil
}
