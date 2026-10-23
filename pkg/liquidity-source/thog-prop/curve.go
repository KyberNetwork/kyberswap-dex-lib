package thogprop

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// Integer operations and their order follow deployed MakerRoutingMath. In
// particular, the volume charge is outside the positive Kelly-risk clamp,
// and the output refund is part of the output solve, before token conversion.
type curve struct {
	input, output                                                                      int
	posted                                                                             uint64
	priceIn, priceOut, inputScale, outputScale, spread, lambda, volumeRate, inputLimit uint256.Int
	variance, gradient                                                                 int256.Int
	entryFixed, entrySlope, entryInventory, exitFixed, exitSlope, exitInventory        uint256.Int
}

func (c *curve) delta(m *arithmetic, n uint256.Int) int256.Int {
	v := m.signed(n)
	quadratic := m.id(m.im(m.im(v, v), c.variance), m.signed(powers[27]))
	linear := m.signed(m.md(n, m.abs(c.gradient), u(500000000000000000), false))
	if c.gradient.IsNegative() {
		linear.Neg(&linear)
	}
	return m.ia(quadratic, linear)
}
func (c *curve) penalty(m *arithmetic, n uint256.Int) uint256.Int {
	if c.lambda.IsZero() {
		return uint256.Int{}
	}
	delta := c.delta(m, n)
	if delta.Sign() <= 0 {
		return uint256.Int{}
	}
	return m.div(m.mul(uint256.Int(delta), c.lambda), u(1000000))
}
func (c *curve) penaltySlope(m *arithmetic, n uint256.Int) uint256.Int {
	if c.lambda.IsZero() {
		return uint256.Int{}
	}
	delta := c.delta(m, n)
	slope := m.id(m.im(m.ia(m.id(m.im(m.im(si(2), m.signed(n)), c.variance), si(1000000000)), m.im(si(2), c.gradient)), m.signed(c.lambda)), si(1000000))
	if (delta.Sign() > 0 || (delta.IsZero() && slope.Sign() > 0)) && slope.Sign() > 0 {
		return uint256.Int(slope)
	}
	return uint256.Int{}
}
func (c *curve) entryCost(m *arithmetic, n uint256.Int) uint256.Int {
	if !c.entryFixed.IsZero() {
		return m.md(n, c.entryFixed, powers[18], true)
	}
	if c.entrySlope.IsZero() {
		return uint256.Int{}
	}
	return m.md(c.entrySlope, m.mul(n, m.add(m.mul(u(2), c.entryInventory), n)), m.mul(u(2), powers[54]), true)
}
func (c *curve) outputForBudget(m *arithmetic, budget uint256.Int) uint256.Int {
	if !c.exitFixed.IsZero() {
		return m.md(budget, powers[18], m.add(powers[18], c.exitFixed), false)
	}
	k := c.exitSlope
	if k.IsZero() {
		return budget
	}
	inventory := c.exitInventory
	maximum := m.div(m.sub(powers[54], u(1)), k)
	if inventory.Gt(&maximum) {
		m.err = ErrUnavailable
		return uint256.Int{}
	}
	denominator := m.mul(u(2), powers[54])
	credit := m.md(k, m.mul(inventory, inventory), denominator, false)
	threshold := m.sub(inventory, credit)
	if budget.Cmp(&threshold) >= 0 {
		return m.add(budget, credit)
	}
	b := m.sub(powers[54], m.mul(k, inventory))
	kBudget := m.mul(k, budget)
	root := sqrtDiscriminantCeil(m, b, kBudget)
	output := m.md(denominator, budget, m.add(b, root), false)
	next := m.add(output, u(1))
	nextCredit := m.md(k, m.mul(next, m.sub(m.mul(u(2), inventory), next)), denominator, false)
	cost := m.sub(next, nextCredit)
	if cost.Cmp(&budget) <= 0 {
		return next
	}
	return output
}

// Computes the exact 512-bit discriminant division without decimal rescaling.
// The two remainders fit: divisor is bounded by sqrt(D) < 2^181 here.
func discriminantDiv(m *arithmetic, b, kBudget, divisor uint256.Int) (q, r uint256.Int) {
	denominator := m.mul(u(2), powers[54])
	q = m.add(m.md(b, b, divisor, false), m.md(denominator, kBudget, divisor, false))
	var r1, r2 uint256.Int
	r1.MulMod(&b, &b, &divisor)
	r2.MulMod(&denominator, &kBudget, &divisor)
	r = m.add(r1, r2)
	if r.Cmp(&divisor) >= 0 {
		q = m.add(q, u(1))
		r = m.sub(r, divisor)
	}
	return
}
func sqrtDiscriminantCeil(m *arithmetic, b, kBudget uint256.Int) uint256.Int {
	if kBudget.IsZero() {
		return b
	}
	var root uint256.Int
	root.Sqrt(&kBudget)
	square := m.mul(root, root)
	if square != kBudget {
		root = m.add(root, u(1))
	}
	x := m.add(b, m.mul(root, sqrt2e54))
	for m.err == nil {
		q, _ := discriminantDiv(m, b, kBudget, x)
		next := m.div(m.add(x, q), u(2))
		if next.Cmp(&x) >= 0 {
			break
		}
		x = next
	}
	q, r := discriminantDiv(m, b, kBudget, x)
	if q != x || !r.IsZero() {
		x = m.add(x, u(1))
	}
	return x
}
func (c *curve) quote(amount uint256.Int, friction bool) (q Quote, err error) {
	q.AmountIn = amount
	q.PostedBlock = c.posted
	if amount.Gt(&c.inputLimit) {
		return q, ErrUnavailable
	}
	m := arithmetic{}
	q.Notional = m.md(amount, c.priceIn, c.inputScale, false)
	if q.Notional.IsZero() {
		return q, m.err
	}
	available := m.md(q.Notional, m.sub(u(1000000), c.spread), u(1000000), false)
	q.Penalty = c.penalty(&m, q.Notional)
	q.VolumeCost = m.md(m.mul(q.Notional, q.Notional), c.volumeRate, powers[27], true)
	entry := c.entryCost(&m, q.Notional)
	for _, charge := range [3]uint256.Int{q.Penalty, q.VolumeCost, entry} {
		if charge.Cmp(&available) >= 0 {
			return q, m.err
		}
		available = m.sub(available, charge)
	}
	output := c.outputForBudget(&m, available)
	q.AmountOut = m.md(output, c.outputScale, c.priceOut, false)
	if friction {
		q.AmountOut = m.md(q.AmountOut, u(9990), u(10000), false)
	}
	return q, m.err
}
func (c *curve) remainingSlope(m *arithmetic, amount uint256.Int) uint256.Int {
	n := m.md(amount, c.priceIn, c.inputScale, false)
	base := m.md(m.sub(u(1000000), c.spread), powers[36], u(1000000), false)
	cost := m.add(m.mul(c.penaltySlope(m, n), powers[18]), m.mul(c.entryFixed, powers[18]))
	cost = m.add(cost, m.mul(m.mul(c.volumeRate, n), u(2000000000)))
	cost = m.add(cost, m.md(c.entrySlope, m.add(c.entryInventory, n), powers[18], true))
	if cost.Lt(&base) {
		return m.sub(base, cost)
	}
	return uint256.Int{}
}
func (c *curve) marginal(amount uint256.Int) (p Fraction, err error) {
	if amount.Gt(&c.inputLimit) {
		return p, ErrUnavailable
	}
	m := arithmetic{}
	slope := c.remainingSlope(&m, amount)
	if slope.IsZero() {
		p.Denominator = u(1)
		return p, m.err
	}
	q, err := c.quote(amount, false)
	if err != nil {
		return p, err
	}
	output := m.md(q.AmountOut, c.priceOut, c.outputScale, false)
	denominator := m.add(powers[36], m.mul(c.exitFixed, powers[18]))
	if !c.exitSlope.IsZero() && output.Lt(&c.exitInventory) {
		refund := m.md(c.exitSlope, m.sub(c.exitInventory, output), powers[18], false)
		if refund.Cmp(&powers[36]) >= 0 {
			return p, ErrUnavailable
		}
		denominator = m.sub(denominator, refund)
	}
	fair := m.md(c.priceIn, powers[36], c.priceOut, false)
	p.Numerator = m.md(fair, slope, denominator, false)
	if p.Numerator.IsZero() {
		p.Denominator = u(1)
	} else if c.inputScale.Cmp(&c.outputScale) >= 0 {
		p.Denominator = m.mul(powers[36], m.div(c.inputScale, c.outputScale))
	} else {
		p.Denominator = m.div(powers[36], m.div(c.outputScale, c.inputScale))
	}
	return p, m.err
}
func (c *curve) solveOutput(target uint256.Int, friction bool) (Quote, error) {
	if target.IsZero() {
		return c.quote(uint256.Int{}, friction)
	}
	m := arithmetic{}
	targetUsd := m.md(target, c.priceOut, c.outputScale, true)
	high := min(max(u(1), m.md(targetUsd, c.inputScale, c.priceIn, true)), c.inputLimit)
	if m.err != nil {
		return Quote{}, m.err
	}
	var low uint256.Int
	q, err := c.quote(high, friction)
	if err != nil {
		return q, err
	}
	for q.AmountOut.Lt(&target) {
		slope := c.remainingSlope(&m, high)
		if m.err != nil {
			return q, m.err
		}
		if slope.IsZero() {
			a, b := low, high
			for {
				gap := m.sub(b, a)
				if gap.CmpUint64(1) <= 0 {
					break
				}
				mid := m.add(a, m.div(gap, u(2)))
				s := c.remainingSlope(&m, mid)
				if m.err != nil {
					return q, m.err
				}
				if s.IsZero() {
					b = mid
				} else {
					a = mid
				}
			}
			qa, e := c.quote(a, friction)
			if e != nil {
				return qa, e
			}
			qb, e := c.quote(b, friction)
			if e != nil {
				return qb, e
			}
			if qb.AmountOut.Gt(&qa.AmountOut) {
				high, q = b, qb
			} else {
				high, q = a, qa
			}
			if q.AmountOut.Lt(&target) {
				return q, ErrUnavailable
			}
			break
		}
		if high == c.inputLimit {
			return q, ErrUnavailable
		}
		low = high
		half := m.div(c.inputLimit, u(2))
		if high.Gt(&half) {
			high = c.inputLimit
		} else {
			high = m.mul(high, u(2))
		}
		q, err = c.quote(high, friction)
		if err != nil {
			return q, err
		}
	}
	for {
		gap := m.sub(high, low)
		if gap.CmpUint64(1) <= 0 {
			break
		}
		mid := m.add(low, m.div(gap, u(2)))
		candidate, e := c.quote(mid, friction)
		if e != nil {
			return candidate, e
		}
		if candidate.AmountOut.Cmp(&target) >= 0 {
			high, q = mid, candidate
		} else {
			low = mid
		}
	}
	return q, m.err
}
