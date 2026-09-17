package lunyafun

import (
	"github.com/holiman/uint256"
)

// A port of LunyaLaunchCP's curve: _quoteBuy and _quoteSell, with the anti-snipe surcharge that sits on
// top of the curve fee. The curve is constant product over virtual reserves - x = virtualQuote +
// reserve, y = virtualToken - sold - and a buy that would take the last of curveSupply is capped there,
// with the unspent input refunded.

var bpsU = uint256.NewInt(bps)

// curveState is what pricing needs from a launch.
type curveState struct {
	reserve      uint256.Int
	sold         uint256.Int
	virtualQuote uint256.Int
	virtualToken uint256.Int
	curveSupply  uint256.Int
	curveFeeBps  uint64
	snipeTaxBps  uint64
	snipeWindow  uint64
	snipeDecay   uint64
	openedAt     uint64
}

// snipeTaxBpsAt is _snipeTaxBps for a recipient the launch does not exempt: the tax decays as
// remaining^decay / window^decay over the window, and is gone once the window has passed.
func (c *curveState) snipeTaxBpsAt(now uint64) uint64 {
	if c.snipeWindow == 0 || c.snipeTaxBps == 0 || now < c.openedAt {
		return 0
	}

	elapsed := now - c.openedAt
	if elapsed >= c.snipeWindow {
		return 0
	}

	remaining := c.snipeWindow - elapsed
	var tax, numerator, denominator uint256.Int
	numerator.SetUint64(c.snipeTaxBps)
	pow(&tax, remaining, c.snipeDecay)
	numerator.Mul(&numerator, &tax)
	pow(&denominator, c.snipeWindow, c.snipeDecay)
	if denominator.IsZero() {
		return 0
	}
	return numerator.Div(&numerator, &denominator).Uint64()
}

func pow(z *uint256.Int, base, exponent uint64) {
	z.SetUint64(1)
	var b uint256.Int
	b.SetUint64(base)
	for range exponent {
		z.Mul(z, &b)
	}
}

// quoteBuy prices a buy of amountIn quote tokens at time now, for a recipient the launch does not
// exempt. It returns the tokens out, the fee, and the input left unspent because the curve ran out.
func (c *curveState) quoteBuy(amountIn *uint256.Int, now uint64) (tokensOut, fee, refund uint256.Int, err error) {
	if amountIn.IsZero() {
		return
	}

	var x, y, remaining uint256.Int
	x.Add(&c.virtualQuote, &c.reserve)
	if c.virtualToken.Lt(&c.sold) || c.curveSupply.Lt(&c.sold) {
		err = ErrArithmetic
		return
	}
	y.Sub(&c.virtualToken, &c.sold)
	remaining.Sub(&c.curveSupply, &c.sold)

	snipeBps := c.snipeTaxBpsAt(now)
	feeBps := c.curveFeeBps + snipeBps
	if feeBps >= bps {
		feeBps = bps - 1
		snipeBps = feeBps - c.curveFeeBps
	}

	var feeBpsU, net uint256.Int
	feeBpsU.SetUint64(feeBps)
	if _, overflow := fee.MulOverflow(amountIn, &feeBpsU); overflow {
		err = ErrArithmetic
		return
	}
	fee.Div(&fee, bpsU)
	net.Sub(amountIn, &fee)

	var numerator, denominator uint256.Int
	if _, overflow := numerator.MulOverflow(&y, &net); overflow {
		err = ErrArithmetic
		return
	}
	denominator.Add(&x, &net)
	if denominator.IsZero() {
		err = ErrArithmetic
		return
	}
	tokensOut.Div(&numerator, &denominator)

	if tokensOut.Lt(&remaining) {
		return
	}

	// the buy takes the last of the curve: it pays for `remaining` and gets the rest back
	tokensOut.Set(&remaining)
	if !y.Gt(&remaining) {
		err = ErrArithmetic
		return
	}
	var netRequired, gross, denom uint256.Int
	if _, overflow := numerator.MulOverflow(&x, &remaining); overflow {
		err = ErrArithmetic
		return
	}
	denom.Sub(&y, &remaining)
	ceilDiv(&netRequired, &numerator, &denom)
	if netRequired.Gt(&net) {
		netRequired.Set(&net)
	}
	if _, overflow := numerator.MulOverflow(&netRequired, bpsU); overflow {
		err = ErrArithmetic
		return
	}
	denom.SetUint64(bps - feeBps)
	ceilDiv(&gross, &numerator, &denom)
	if gross.Gt(amountIn) {
		gross.Set(amountIn)
	}
	fee.Sub(&gross, &netRequired)
	refund.Sub(amountIn, &gross)

	return
}

// quoteSell prices a sell of tokensIn launched tokens, returning the quote out and the fee.
func (c *curveState) quoteSell(tokensIn *uint256.Int) (amountOut, fee uint256.Int, err error) {
	if tokensIn.IsZero() {
		return
	}
	if tokensIn.Gt(&c.sold) {
		err = ErrMoreThanSold
		return
	}

	var x, y, numerator, denominator, gross uint256.Int
	x.Add(&c.virtualQuote, &c.reserve)
	if c.virtualToken.Lt(&c.sold) {
		err = ErrArithmetic
		return
	}
	y.Sub(&c.virtualToken, &c.sold)

	if _, overflow := numerator.MulOverflow(&x, tokensIn); overflow {
		err = ErrArithmetic
		return
	}
	denominator.Add(&y, tokensIn)
	if denominator.IsZero() {
		err = ErrArithmetic
		return
	}
	gross.Div(&numerator, &denominator)
	if gross.Gt(&c.reserve) {
		gross.Set(&c.reserve) // rounding-dust guard, as on-chain
	}

	var feeBpsU uint256.Int
	feeBpsU.SetUint64(c.curveFeeBps)
	if _, overflow := fee.MulOverflow(&gross, &feeBpsU); overflow {
		err = ErrArithmetic
		return
	}
	fee.Div(&fee, bpsU)
	amountOut.Sub(&gross, &fee)

	return
}

// ceilDiv is OpenZeppelin's Math.ceilDiv.
func ceilDiv(z, a, b *uint256.Int) {
	if a.IsZero() {
		z.Clear()
		return
	}
	z.SubUint64(a, 1).Div(z, b).AddUint64(z, 1)
}
