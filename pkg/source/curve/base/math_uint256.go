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

	u256One = uint256.NewInt(1)
	u256Two = uint256.NewInt(2)
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
