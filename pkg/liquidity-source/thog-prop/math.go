package thogprop

import (
	"strings"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func (s *State) covariance(a, b int) int64 {
	return int64(int16(s.small(covarianceField(covarianceIndex(a, b)))))
}

// price decodes a token's USD price, scaled by 1e18. A stored price stands for
// the interval [mantissa, mantissa+1) at its exponent: the pool values what it
// receives at the lower end and what it pays at the upper end.
func (s *State) price(index int, upper bool) uint256.Int {
	packed := s.small(priceField(index))
	exponent := packed >> 56
	mantissa := packed & ((1 << 56) - 1)
	if mantissa < minPriceMantissa || exponent > 18 {
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
	if n < 1 || n > maxTokens || c < 1 || c > n || s.small(controlLayoutVersion) != supportedLayout || s.small(controlTokenCount) != uint64(n) || s.small(controlCategoryCount) != uint64(c) || s.small(metadataVersions) != supportedVersions || s.small(metadataGeneration) == 0 {
		return ErrInvalidState
	}
	for i, t := range s.Tokens {
		if s.shareBased(i) {
			// The page reports a share-based token at the pool's stored multiplier.
			m := arithmetic{}
			if balance := s.amountForUnits(&m, i, s.Shares[i].Shares); m.err != nil || balance != s.Balances[i] {
				return ErrInvalidState
			}
		} else if !s.Shares[i].Shares.IsZero() {
			return ErrInvalidState
		}
		catalogID := int(s.small(tokenField(i, tokenCatalogID)))
		if s.small(registryIndex(catalogID)) != uint64(i+1) {
			return ErrInvalidState
		}
		if t.Decimals > 18 || int(t.Category) >= c || !common.IsHexAddress(t.Address) || t.Address != strings.ToLower(t.Address) || t.Address == "0x0000000000000000000000000000000000000000" || s.small(tokenField(i, tokenDecimals)) != uint64(t.Decimals) || s.small(tokenField(i, tokenCategory)) != uint64(t.Category) {
			return ErrInvalidState
		}
		for j := 0; j < i; j++ {
			if t.Address == s.Tokens[j].Address {
				return ErrInvalidState
			}
		}
	}
	for i := 0; i < c; i++ {
		anchor := s.small(categoryField(i, categoryAnchor))
		if anchor >= uint64(n) || int(s.Tokens[anchor].Category) != i {
			return ErrInvalidState
		}
	}
	return nil
}
func (s *State) inventory(m *arithmetic, balances *[maxTokens]uint256.Int, index int, native, bounded bool) uint256.Int {
	balance := balances[index]
	if native && s.Tokens[index].Address == wmon {
		balance = m.add(balance, s.NativeBalance)
	}
	value := m.md(balance, s.price(index, false), powers[s.Tokens[index].Decimals], false)
	if bounded && value.Gt(&maxInventory) {
		m.err = ErrOverflow
	}
	return value
}
func (s *State) wrapper(m *arithmetic, balances *[maxTokens]uint256.Int, index int, receiving bool) (fixed, slope, inventory uint256.Int) {
	kind := s.small(tokenField(index, tokenWrapperKind))
	if kind == wrapperNone {
		return
	}
	coefficient := s.field(wrapperCoefficient(index))
	if kind == wrapperFixed {
		fixed = coefficient
		return
	}
	target := m.mul(s.field(categoryTarget(int(s.Tokens[index].Category))), powers[12])
	if target.IsZero() {
		m.err = ErrUnavailable
		return
	}
	inventory = s.inventory(m, balances, index, false, true)
	if receiving {
		slope = m.md(coefficient, powers[36], target, true)
	} else {
		slope = m.md(m.mul(coefficient, s.field(tokenField(index, tokenWrapperRefund))), powers[36], m.mul(target, u(1000)), false)
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
	return s.prepareWithAgeCheck(i, o, block, true)
}

func (s *State) prepareWithAgeCheck(i, o int, block uint64, checkAge bool) (c curve, err error) {
	if i < 0 || o < 0 || i >= len(s.Tokens) || o >= len(s.Tokens) || i == o {
		return c, ErrInvalidToken
	}
	if block < s.BlockNumber || s.small(controlPaused) != 0 || s.small(controlReady) != controlAllReady || s.small(controlSellSide(i)) == 0 || s.small(controlBuySide(o)) == 0 {
		return c, ErrUnavailable
	}
	posted := s.small(controlBlock)
	age := s.small(controlMaxAge)
	if posted == 0 || posted > block || (checkAge && (age == 0 || block-posted > age)) {
		return c, ErrUnavailable
	}
	m := arithmetic{}
	c.balances = s.pairBalances(&m, i, o)
	c.inputDust, c.outputDust = s.transferDust(&m, i), s.transferDust(&m, o)
	if m.err != nil {
		return c, m.err
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
	c.lambda = s.field(controlLambda)
	c.volumeRate = s.field(tokenField(i, tokenVolumeRate))
	a, b := int(s.Tokens[i].Category), int(s.Tokens[o].Category)
	// Spread: the input token's bid, the output token's ask, both categories, the
	// pair, and the widening for each block since the price update.
	c.spread = m.add(m.add(s.field(tokenField(i, tokenBidSpread)), s.field(tokenField(o, tokenAskSpread))), m.add(s.field(categoryField(a, categorySpread)), s.field(categoryField(b, categorySpread))))
	c.spread = m.add(c.spread, m.add(s.field(pairSpreadField(pairIndex(i, o))), m.mul(u(block-posted), s.field(controlWiden))))
	if c.spread.CmpUint64(1000000) >= 0 {
		return c, ErrUnavailable
	}
	if !c.lambda.IsZero() && a != b {
		c.variance = si(s.covariance(a, a) + s.covariance(b, b) - 2*s.covariance(a, b))
		var diff [maxTokens]int64
		var exposure [maxTokens]int256.Int
		for category := 0; category < s.CategoryCount; category++ {
			diff[category] = s.covariance(a, category) - s.covariance(b, category)
			if diff[category] == 0 {
				continue
			}
			offset := int64(s.small(categoryField(category, categoryOffset)))
			anchor := int(s.small(categoryField(category, categoryAnchor)))
			exposure[category] = m.id(m.im(si(offset), m.signed(s.price(anchor, false))), si(1000000))
		}
		for index, t := range s.Tokens {
			category := int(t.Category)
			if diff[category] != 0 {
				exposure[category] = m.ia(exposure[category], m.signed(s.inventory(&m, &c.balances, index, true, true)))
			}
		}
		for category := 0; category < s.CategoryCount; category++ {
			if diff[category] == 0 {
				continue // This category's exposure was not evaluated and contributes zero.
			}
			c.gradient = m.ia(c.gradient, m.id(m.im(exposure[category], si(diff[category])), si(1000000000)))
		}
	}
	c.entryFixed, c.entrySlope, c.entryInventory = s.wrapper(&m, &c.balances, i, true)
	c.exitFixed, c.exitSlope, c.exitInventory = s.wrapper(&m, &c.balances, o, false)
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
	n := packed.Uint64()
	if n != 0xffffff && (n>>16 > 240 || (n>>16 != 0 && n&65535 < 32768)) {
		m.err = ErrInvalidState
		return uint256.Int{}
	}
	value := capValue(packed)
	if quote && !value.IsZero() {
		value = m.sub(m.md(value, u(90), u(100), true), u(1))
	}
	return value
}
func (s *State) checkInventory(c *curve, input, output uint256.Int, quote bool) error {
	m := arithmetic{}
	if !quote {
		swap, err := s.settlement(c, input, output)
		if err != nil {
			return err
		}
		input = m.sub(swap.InputBalance, c.balances[c.input])
		output = m.sub(c.balances[c.output], swap.OutputBalance)
	} else if !output.IsZero() {
		output = m.add(output, c.outputDust)
	}
	limit := min(max(capLimit(&m, s.field(tokenCap(c.input)), quote), c.balances[c.input]), s.amountForUnits(&m, c.input, maxInventory))
	if m.err != nil {
		return m.err
	}
	if output.Gt(&c.balances[c.output]) || c.balances[c.input].Gt(&limit) {
		return ErrBalance
	}
	remaining := m.sub(limit, c.balances[c.input])
	if input.Gt(&remaining) {
		return ErrBalance
	}
	var value, beforeValue uint256.Int
	category := s.Tokens[c.input].Category
	for i, t := range s.Tokens {
		if t.Category != category {
			continue
		}
		balance := c.balances[i]
		if t.Address == wmon {
			balance = m.add(balance, s.NativeBalance)
		}
		price := s.price(i, false)
		tokenValue := m.md(balance, price, powers[t.Decimals], false)
		beforeValue = m.add(beforeValue, tokenValue)
		if i == c.input {
			balance = m.add(balance, input)
		}
		if i == c.output {
			balance = m.sub(balance, output)
		}
		if i == c.input || i == c.output {
			tokenValue = m.md(balance, price, powers[t.Decimals], false)
		}
		value = m.add(value, tokenValue)
	}
	categoryLimit := max(capLimit(&m, s.field(categoryCap(int(category))), quote), beforeValue)
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
	return s.QuoteExactInputAt(i, o, amount, s.BlockNumber)
}

// QuoteExactInputAt advances only the quote block, without predicting updates,
// trades or issuer changes. The source snapshot block remains unchanged.
func (s *State) QuoteExactInputAt(i, o int, amount uint256.Int, block uint64) (Quote, error) {
	c, err := s.prepare(i, o, block)
	if err != nil {
		return Quote{}, err
	}
	return s.quoteReadOnly(&c, amount)
}

func (s *State) quoteReadOnly(c *curve, amount uint256.Int) (Quote, error) {
	q, err := c.quote(amount, 0)
	if err != nil {
		return q, err
	}
	if q.AmountOut.IsZero() {
		return q, ErrUnavailable
	}
	return q, s.checkInventory(c, amount, q.AmountOut, true)
}
func (s *State) QuoteExecution(i, o int, amount uint256.Int, ctx ExecutionContext) (Quote, error) {
	c, err := s.prepare(i, o, ctx.BlockNumber)
	if err != nil {
		return Quote{}, err
	}
	friction := ctx.friction(c.posted == ctx.BlockNumber)
	// Sell applies friction after output dust; Buy applies it before dust.
	q, err := c.quote(amount, 0)
	if err == nil && friction != 0 {
		m := arithmetic{}
		q.AmountOut = applyExecutionFriction(&m, q.AmountOut, friction)
		err = m.err
	}
	if err != nil {
		return q, err
	}
	if q.AmountOut.IsZero() {
		return q, ErrUnavailable
	}
	return q, s.checkInventory(&c, amount, q.AmountOut, false)
}
func (s *State) QuoteExactOutput(i, o int, target uint256.Int, ctx ExecutionContext) (Quote, error) {
	c, err := s.prepare(i, o, ctx.BlockNumber)
	if err != nil {
		return Quote{}, err
	}
	buy := saturatingSub(s.Balances[o], c.outputDust)
	if target.Gt(&buy) {
		return Quote{}, ErrBalance
	}
	friction := ctx.friction(c.posted == ctx.BlockNumber)
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
	q, err := c.quote(amount, 0)
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
	buy = saturatingSub(s.Balances[o], c.outputDust)
	limit := min(capLimit(&m, s.field(tokenCap(i)), true), s.amountForUnits(&m, i, maxInventory))
	if !s.Balances[o].IsZero() && s.Balances[i].Lt(&limit) {
		sell = min(c.inputLimit, m.sub(limit, s.Balances[i]))
	}
	return sell, buy, m.err
}
