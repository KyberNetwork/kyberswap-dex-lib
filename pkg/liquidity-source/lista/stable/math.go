package stable

import (
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// maxCoins bounds the stack arrays; lista pools have 2 coins (checkPriceDiff rejects any other count).
const maxCoins = 8

var (
	precision      = big256.BONE
	feeDenominator = big256.TenPow(10)
)

// getDy is curve/base's GetDy: the output of dx of coin i in coin j, and the fee, both in coin j.
// dy is 0 when the swap gives nothing (curve/base's CalcAmountOut rejects that).
func (t *PoolSimulator) getDy(i, j int, dx, dy, fee *uint256.Int) error {
	if len(t.reserves) > maxCoins {
		return ErrInvalidValue
	}
	var xpArr [maxCoins]uint256.Int
	xp := xpArr[:len(t.reserves)]
	if err := t.xpMem(t.reserves, xp); err != nil {
		return err
	}
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x uint256.Int
	if err := mulDivAdd(&x, dx, &t.rates[i], &xp[i]); err != nil {
		return err
	}
	a, err := t._A()
	if err != nil {
		return err
	}
	var y uint256.Int
	if err := t.getY(i, j, &x, xp, &a, nil, &y); err != nil {
		return err
	}

	// dy: uint256 = xp[j] - y - 1
	if !xp[j].Gt(&y) {
		dy.Clear()
		fee.Clear()
		return nil
	}
	dy.Sub(&xp[j], &y)
	dy.SubUint64(dy, 1)
	// fee: uint256 = self.fee * dy / FEE_DENOMINATOR
	if _, overflow := fee.MulDivOverflow(&t.swapFee, dy, feeDenominator); overflow {
		return errOverflow
	}
	// (dy - fee) * PRECISION / rates[j], fee * PRECISION / rates[j]
	dy.Sub(dy, fee)
	if _, overflow := dy.MulDivOverflow(dy, precision, &t.rates[j]); overflow {
		return errOverflow
	}
	if _, overflow := fee.MulDivOverflow(fee, precision, &t.rates[j]); overflow {
		return errOverflow
	}
	return nil
}

// getDyWithoutFee is lista's get_dy_without_fee on balances whose xp, A and D are given.
func (t *PoolSimulator) getDyWithoutFee(xp []uint256.Int, a, d *uint256.Int, i, j int, dx, dy *uint256.Int) error {
	// x = xp[i] + dx * rates[i] / PRECISION
	var x uint256.Int
	if err := mulDivAdd(&x, dx, &t.rates[i], &xp[i]); err != nil {
		return err
	}
	var y uint256.Int
	if err := t.getY(i, j, &x, xp, a, d, &y); err != nil {
		return err
	}

	// dy = (xp[j] - y - 1) * PRECISION / rates[j], converting XP → raw token units (matches
	// get_dy_without_fee on-chain). It rounds toward zero, as the big.Int version did: a negative dy
	// that rounds to 0 is 0, and one that does not is an error (the big.Int version panicked).
	if xp[j].Gt(&y) {
		dy.Sub(&xp[j], &y)
		dy.SubUint64(dy, 1)
		if _, overflow := dy.MulDivOverflow(dy, precision, &t.rates[j]); overflow {
			return errOverflow
		}
		return nil
	}
	var negative uint256.Int
	negative.Sub(&y, &xp[j])
	negative.AddUint64(&negative, 1)
	if negative.MulDivOverflow(&negative, precision, &t.rates[j]); !negative.IsZero() {
		return errNegativeDy
	}
	dy.Clear()
	return nil
}

// xpMem: xp[i] = rates[i] * balances[i] / PRECISION
func (t *PoolSimulator) xpMem(balances, xp []uint256.Int) error {
	for i := range xp {
		if _, overflow := xp[i].MulDivOverflow(&t.rates[i], &balances[i], precision); overflow {
			return errOverflow
		}
	}
	return nil
}

// mulDivAdd sets z = x * rate / PRECISION + add.
func mulDivAdd(z, x, rate, add *uint256.Int) error {
	if _, overflow := z.MulDivOverflow(x, rate, precision); overflow {
		return errOverflow
	}
	if _, overflow := z.AddOverflow(z, add); overflow {
		return errOverflow
	}
	return nil
}

// _A is A * A_PRECISION, interpolated while A ramps. A ramp that has not started yet is an error:
// on chain `block.timestamp - initial_A_time` underflows (the big.Int version extrapolated backwards).
func (t *PoolSimulator) _A() (uint256.Int, error) {
	now := time.Now().Unix()
	if t.futureATime <= now {
		return t.futureA, nil
	}
	if now < t.initialATime {
		return uint256.Int{}, errRampNotStarted
	}
	var a, delta uint256.Int
	elapsed, duration := uint256.NewInt(uint64(now-t.initialATime)), uint256.NewInt(uint64(t.futureATime-t.initialATime))
	if t.futureA.Gt(&t.initialA) {
		delta.Sub(&t.futureA, &t.initialA)
		delta.Div(delta.Mul(&delta, elapsed), duration)
		a.Add(&t.initialA, &delta)
	} else {
		delta.Sub(&t.initialA, &t.futureA)
		delta.Div(delta.Mul(&delta, elapsed), duration)
		a.Sub(&t.initialA, &delta)
	}
	return a, nil
}

// getD is curve/base's getD: D_P = D_P * D / (x * N_COINS + 1), and D_P * D must fit in uint256.
// https://github.com/curvefi/curve-contract/blob/d4e8589ac92c4019b3064b2f3a8a87dbc3281b46/contracts/pool-templates/base/SwapTemplateBase.vy#L217
func (t *PoolSimulator) getD(xp []uint256.Int, a, d *uint256.Int) error {
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

	var nA, nASDivAP, nASubAP, nPlus1, dP, prevD, num, den, tmp uint256.Int
	nPlus1.AddUint64(&t.nCoins, 1)
	if _, overflow := nA.MulOverflow(a, &t.nCoins); overflow {
		return errOverflow
	}
	// Ann * S / A_PRECISION and Ann - A_PRECISION
	if _, overflow := nASDivAP.MulDivOverflow(&nA, &s, &t.aPrecision); overflow {
		return errOverflow
	}
	if nA.Lt(&t.aPrecision) {
		return errUnderflow
	}
	nASubAP.Sub(&nA, &t.aPrecision)

	d.Set(&s)
	for range base.MaxLoopLimit {
		dP.Set(d)
		for j := range xp {
			if _, overflow := tmp.MulOverflow(&dP, d); overflow {
				return base.ErrDDoesNotConverge
			}
			if _, overflow := den.MulOverflow(&xp[j], &t.nCoins); overflow {
				return errOverflow
			}
			dP.Div(&tmp, den.AddUint64(&den, 1))
		}
		prevD.Set(d)

		// D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)
		num.Mul(&dP, &t.nCoins)
		if _, overflow := num.AddOverflow(&num, &nASDivAP); overflow {
			return errOverflow
		}
		if _, overflow := num.MulOverflow(&num, d); overflow {
			return errOverflow
		}
		if _, overflow := tmp.MulDivOverflow(&nASubAP, d, &t.aPrecision); overflow {
			return errOverflow
		}
		den.Mul(&dP, &nPlus1)
		if _, overflow := den.AddOverflow(&den, &tmp); overflow || den.IsZero() {
			return errOverflow
		}
		d.Div(&num, &den)

		if withinOne(d, &prevD) {
			return nil
		}
	}
	return base.ErrDDoesNotConverge
}

// getY is curve/base's GetY with A given, and D too when d is not nil.
// https://github.com/curvefi/curve-contract/blob/d4e8589ac92c4019b3064b2f3a8a87dbc3281b46/contracts/pool-templates/base/SwapTemplateBase.vy#L408
func (t *PoolSimulator) getY(i, j int, x *uint256.Int, xp []uint256.Int, a, d, y *uint256.Int) error {
	n := len(xp)
	if i == j {
		return base.ErrTokenFromEqualsTokenTo
	}
	if i >= n && j >= n {
		return base.ErrTokenIndexesOutOfRange
	}
	var dv uint256.Int
	if d != nil {
		dv.Set(d)
	} else if err := t.getD(xp, a, &dv); err != nil {
		return err
	}

	var c, s, nA, xk, tmp uint256.Int
	c.Set(&dv)
	if _, overflow := nA.MulOverflow(a, &t.nCoins); overflow {
		return errOverflow
	}
	for k := range n {
		if k == i {
			xk.Set(x)
		} else if k != j {
			xk.Set(&xp[k])
		} else {
			continue
		}
		if xk.IsZero() {
			return base.ErrZero
		}
		if _, overflow := s.AddOverflow(&s, &xk); overflow {
			return errOverflow
		}
		// c = c * D / (_x * N_COINS)
		if _, overflow := tmp.MulOverflow(&xk, &t.nCoins); overflow {
			return errOverflow
		}
		if _, overflow := c.MulDivOverflow(&c, &dv, &tmp); overflow {
			return errOverflow
		}
	}
	if nA.IsZero() {
		return base.ErrZero
	}
	// c = c * D * A_PRECISION / (Ann * N_COINS)
	if _, overflow := c.MulOverflow(&c, &dv); overflow {
		return errOverflow
	}
	if _, overflow := tmp.MulOverflow(&nA, &t.nCoins); overflow {
		return errOverflow
	}
	if _, overflow := c.MulDivOverflow(&c, &t.aPrecision, &tmp); overflow {
		return errOverflow
	}
	// b = S + D * A_PRECISION / Ann
	var b uint256.Int
	if _, overflow := b.MulDivOverflow(&dv, &t.aPrecision, &nA); overflow {
		return errOverflow
	}
	if _, overflow := b.AddOverflow(&b, &s); overflow {
		return errOverflow
	}

	var yPrev, num, den uint256.Int
	y.Set(&dv)
	for range base.MaxLoopLimit {
		yPrev.Set(y)
		// y = (y*y + c) / (2 * y + b - D)
		den.Lsh(y, 1)
		if _, overflow := den.AddOverflow(&den, &b); overflow {
			return errOverflow
		}
		if !den.Gt(&dv) {
			return base.ErrDenominatorZero
		}
		den.Sub(&den, &dv)
		if _, overflow := num.MulOverflow(y, y); overflow {
			return errOverflow
		}
		if _, overflow := num.AddOverflow(&num, &c); overflow {
			return errOverflow
		}
		y.Div(&num, &den)

		if withinOne(y, &yPrev) {
			return nil
		}
	}
	return base.ErrAmountOutNotConverge
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
