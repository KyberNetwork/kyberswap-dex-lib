package thogprop

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func (s *State) field(slot, offset, width int) uint256.Int {
	return bits(&s.Words[slot], uint(offset), uint(width))
}
func (s *State) small(slot, offset, width int) uint64 {
	n := s.field(slot, offset, width)
	return n.Uint64()
}
func (s *State) packed(slot, index, width, perWord int) uint256.Int {
	return s.field(slot+index/perWord, index%perWord*width, width)
}
func (s *State) tokenField(index, offset, width int) uint256.Int {
	return s.field(9+index/3, index%3*75+offset, width)
}
func (s *State) covariance(a, b int) int64 {
	if a > b {
		a, b = b, a
	}
	i := b*(b+1)/2 + a
	n := s.packed(40, i, 16, 16)
	return int64(int16(n.Uint64()))
}
func (s *State) price(index int, upper bool) uint256.Int {
	n := s.packed(21, index, 64, 4)
	packed := n.Uint64()
	exponent := packed >> 56
	mantissa := packed & ((1 << 56) - 1)
	if mantissa < 2000000 || exponent > 18 {
		return uint256.Int{}
	}
	if upper {
		mantissa++
	}
	var z uint256.Int
	p := u(mantissa)
	z.Mul(&p, &powers[18-exponent])
	return z
}
func (s *State) validate() error {
	n := len(s.Tokens)
	c := s.CategoryCount
	if n < 1 || n > 32 || c < 1 || c > n || s.small(2, 0, 8) != 7 || s.small(2, 8, 6) != uint64(n) || s.small(2, 14, 6) != uint64(c) || s.small(8, 160, 24) != (7|7<<8|1<<16) || s.small(8, 184, 32) == 0 {
		return ErrInvalidState
	}
	for i, t := range s.Tokens {
		catalogID := int(s.small(9+i/3, i%3*75, 5))
		if s.small(20, catalogID*6, 6) != uint64(i+1) {
			return ErrInvalidState
		}
		if t.Decimals > 18 || int(t.Category) >= c || t.Address == "" || s.small(9+i/3, i%3*75+5, 5) != uint64(t.Decimals) || s.small(9+i/3, i%3*75+10, 5) != uint64(t.Category) {
			return ErrInvalidState
		}
		for j := 0; j < i; j++ {
			if t.Address == s.Tokens[j].Address {
				return ErrInvalidState
			}
		}
	}
	for i := 0; i < c; i++ {
		anchor := s.small(29+i/3, i%3*85+80, 5)
		if anchor >= uint64(n) || int(s.Tokens[anchor].Category) != i {
			return ErrInvalidState
		}
	}
	return nil
}
func (s *State) inventory(m *arithmetic, index int, native, bounded bool) uint256.Int {
	balance := s.Balances[index]
	if native && s.Tokens[index].Address == wmon {
		balance = m.add(balance, s.NativeBalance)
	}
	value := m.md(balance, s.price(index, false), powers[s.Tokens[index].Decimals], false)
	if bounded && value.Gt(&maxInventory) {
		m.err = ErrOverflow
	}
	return value
}
func (s *State) wrapper(m *arithmetic, index int, receiving bool) (fixed, slope, inventory uint256.Int) {
	kind := s.tokenField(index, 31, 2)
	if kind.IsZero() {
		return
	}
	coefficient := s.packed(104, index, 64, 4)
	if kind.Uint64() == 1 {
		fixed = coefficient
		return
	}
	target := m.mul(s.packed(112, int(s.Tokens[index].Category), 64, 4), powers[12])
	if target.IsZero() {
		m.err = ErrUnavailable
		return
	}
	inventory = s.inventory(m, index, false, true)
	if receiving {
		slope = m.md(coefficient, powers[36], target, true)
	} else {
		slope = m.md(m.mul(coefficient, s.tokenField(index, 33, 10)), powers[36], m.mul(target, u(1000)), false)
		if !slope.IsZero() {
			maximum := m.div(m.sub(powers[54], u(1)), slope)
			if inventory.Gt(&maximum) {
				m.err = ErrUnavailable
			}
		}
	}
	return
}
func (s *State) prepare(i, o int, block uint64) (c curve, err error) {
	if i < 0 || o < 0 || i >= len(s.Tokens) || o >= len(s.Tokens) || i == o {
		return c, ErrInvalidToken
	}
	if block < s.BlockNumber || s.small(2, 132, 1) != 0 || s.small(2, 133, 3) != 7 || s.small(2, 68+2*i, 1) == 0 || s.small(2, 69+2*o, 1) == 0 {
		return c, ErrUnavailable
	}
	posted := s.small(2, 36, 32)
	age := s.small(2, 138, 8)
	if posted == 0 || posted > block || age == 0 || block-posted > age {
		return c, ErrUnavailable
	}
	c.input = i
	c.output = o
	c.posted = posted
	c.priceIn = s.price(i, false)
	c.priceOut = s.price(o, true)
	if c.priceIn.IsZero() || c.priceOut.IsZero() {
		return c, ErrUnavailable
	}
	c.inputScale = powers[s.Tokens[i].Decimals]
	c.outputScale = powers[s.Tokens[o].Decimals]
	c.lambda = s.field(2, 162, 32)
	c.volumeRate = s.tokenField(i, 59, 16)
	a, b := int(s.Tokens[i].Category), int(s.Tokens[o].Category)
	low, high := i, o
	if low > high {
		low, high = high, low
	}
	pair := high*(high-1)/2 + low
	m := arithmetic{}
	c.spread = m.add(m.add(s.tokenField(i, 15, 16), s.tokenField(o, 43, 16)), m.add(s.field(29+a/3, a%3*85+64, 16), s.field(29+b/3, b%3*85+64, 16)))
	c.spread = m.add(c.spread, m.add(s.packed(73, pair, 16, 16), m.mul(u(block-posted), s.field(2, 146, 16))))
	if c.spread.CmpUint64(1000000) >= 0 {
		return c, ErrUnavailable
	}
	if !c.lambda.IsZero() && a != b {
		c.variance = si(s.covariance(a, a) + s.covariance(b, b) - 2*s.covariance(a, b))
		var diff [32]int64
		var exposure [32]int256.Int
		for category := 0; category < s.CategoryCount; category++ {
			diff[category] = s.covariance(a, category) - s.covariance(b, category)
			if diff[category] == 0 {
				continue
			}
			offset := int64(s.small(29+category/3, category%3*85, 64))
			anchor := int(s.small(29+category/3, category%3*85+80, 5))
			exposure[category] = m.id(m.im(si(offset), m.signed(s.price(anchor, false))), si(1000000))
		}
		for index, t := range s.Tokens {
			category := int(t.Category)
			if diff[category] != 0 {
				exposure[category] = m.ia(exposure[category], m.signed(s.inventory(&m, index, true, true)))
			}
		}
		for category := 0; category < s.CategoryCount; category++ {
			c.gradient = m.ia(c.gradient, m.id(m.im(exposure[category], si(diff[category])), si(1000000000)))
		}
	}
	c.entryFixed, c.entrySlope, c.entryInventory = s.wrapper(&m, i, true)
	c.exitFixed, c.exitSlope, c.exitInventory = s.wrapper(&m, o, false)
	c.inputLimit = m.sub(m.md(m.add(maxNotional, u(1)), c.inputScale, c.priceIn, true), u(1))
	return c, m.err
}
func capValue(packed uint256.Int) uint256.Int {
	if packed.CmpUint64(0xffffff) == 0 {
		return maxUint
	}
	n := packed.Uint64()
	z := u(n & 65535)
	z.Lsh(&z, uint(n>>16))
	return z
}
func capLimit(m *arithmetic, packed uint256.Int, quote bool) uint256.Int {
	value := capValue(packed)
	if quote && !value.IsZero() {
		value = m.sub(m.md(value, u(90), u(100), true), u(1))
	}
	return value
}
func (s *State) checkInventory(c *curve, input, output uint256.Int, quote bool) error {
	m := arithmetic{}
	limit := capLimit(&m, s.packed(120, c.input, 24, 10), quote)
	if output.Gt(&s.Balances[c.output]) || s.Balances[c.input].Gt(&limit) {
		return ErrBalance
	}
	remaining := m.sub(limit, s.Balances[c.input])
	if input.Gt(&remaining) {
		return ErrBalance
	}
	var value uint256.Int
	category := s.Tokens[c.input].Category
	for i, t := range s.Tokens {
		if t.Category != category {
			continue
		}
		balance := s.Balances[i]
		if t.Address == wmon {
			balance = m.add(balance, s.NativeBalance)
		}
		if i == c.input {
			balance = m.add(balance, input)
		}
		if i == c.output {
			balance = m.sub(balance, output)
		}
		value = m.add(value, m.md(balance, s.price(i, false), powers[t.Decimals], false))
	}
	categoryLimit := capLimit(&m, s.packed(124, int(category), 24, 10), quote)
	if m.err != nil {
		return m.err
	}
	if value.Gt(&categoryLimit) {
		return ErrBalance
	}
	return nil
}

// QuoteExactInput matches makerQuoteExactInput at the captured block, including
// the contract's strict 90% quote caps. It performs no network reads.
func (s *State) QuoteExactInput(i, o int, amount uint256.Int) (Quote, error) {
	c, err := s.prepare(i, o, s.BlockNumber)
	if err != nil {
		return Quote{}, err
	}
	q, err := c.quote(amount, false)
	if err != nil {
		return q, err
	}
	if q.AmountOut.IsZero() {
		return q, ErrUnavailable
	}
	return q, s.checkInventory(&c, amount, q.AmountOut, true)
}
func (s *State) QuoteExecution(i, o int, amount uint256.Int, ctx ExecutionContext) (Quote, error) {
	c, err := s.prepare(i, o, ctx.BlockNumber)
	if err != nil {
		return Quote{}, err
	}
	friction, err := ctx.friction()
	if err != nil {
		return Quote{}, err
	}
	q, err := c.quote(amount, friction)
	if err != nil {
		return q, err
	}
	if q.AmountOut.IsZero() {
		return q, ErrUnavailable
	}
	return q, s.checkInventory(&c, amount, q.AmountOut, false)
}
func (ctx ExecutionContext) friction() (bool, error) {
	m := arithmetic{}
	threshold := m.mul(u(2), ctx.BaseFee)
	return ctx.AuctionHandlerWarm || ctx.GasPrice.Gt(&threshold), m.err
}
func (s *State) QuoteExactOutput(i, o int, target uint256.Int, ctx ExecutionContext) (Quote, error) {
	c, err := s.prepare(i, o, ctx.BlockNumber)
	if err != nil {
		return Quote{}, err
	}
	if target.Gt(&s.Balances[o]) {
		return Quote{}, ErrBalance
	}
	friction, err := ctx.friction()
	if err != nil {
		return Quote{}, err
	}
	q, err := c.solveOutput(target, friction)
	if err != nil {
		return q, err
	}
	q.AmountOut = target
	if !target.IsZero() {
		err = s.checkInventory(&c, q.AmountIn, target, false)
	}
	return q, err
}
func (s *State) MarginalPrice(i, o int, amount uint256.Int) (Fraction, error) {
	c, err := s.prepare(i, o, s.BlockNumber)
	if err != nil {
		return Fraction{}, err
	}
	q, err := c.quote(amount, false)
	if err != nil {
		return Fraction{}, err
	}
	if err = s.checkInventory(&c, amount, q.AmountOut, true); err != nil {
		return Fraction{}, err
	}
	return c.marginal(amount)
}
func (s *State) Limits(i, o int) (sell, buy uint256.Int, err error) {
	c, err := s.prepare(i, o, s.BlockNumber)
	if err != nil {
		return sell, buy, err
	}
	m := arithmetic{}
	buy = s.Balances[o]
	limit := capLimit(&m, s.packed(120, i, 24, 10), true)
	if !buy.IsZero() && s.Balances[i].Lt(&limit) {
		sell = min(c.inputLimit, m.sub(limit, s.Balances[i]))
	}
	return sell, buy, m.err
}
