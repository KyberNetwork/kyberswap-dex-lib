package slyngfun

import (
	"github.com/holiman/uint256"
)

// A port of Launchpad._buy and Launchpad.sell: a constant product over virtual reserves,
//
//	k = (virtualQuote + quoteReserve) * tokenReserve,
//
// held constant across trades, with a 1% trade fee on the quote leg and, for the first thirty
// seconds of a curve's life, an opening surcharge that is withheld from a buy and folded into the
// curve's own reserve. Every division truncates, exactly as on chain.

var bpsU = uint256.NewInt(bps)

// curveState is what pricing needs from a curve.
type curveState struct {
	quoteReserve     uint256.Int
	tokenReserve     uint256.Int
	virtualQuote     uint256.Int
	graduationTarget uint256.Int
	createdAt        uint64
	tradeFeeBps      uint64
	snipeBps         uint64
	snipeWindow      uint64
}

// surchargeBpsAt is Launchpad.snipeSurchargeBps: flat across the window, then nothing. A clock
// behind the curve's creation is treated as inside the window, which is the side that never
// overstates the output.
func (c *curveState) surchargeBpsAt(now uint64) uint64 {
	if now < c.createdAt || now-c.createdAt < c.snipeWindow {
		return c.snipeBps
	}
	return 0
}

// quoteBuy prices a buy of amountIn quote at time now. It returns the tokens out, the trade fee
// and the opening surcharge, both taken from amountIn. The launchpad never refunds a buy: every
// wei of input past the fee and surcharge is spent on the curve, however little it buys.
func (c *curveState) quoteBuy(amountIn *uint256.Int, now uint64) (tokensOut, fee, surcharge uint256.Int, err error) {
	if amountIn.IsZero() {
		err = ErrZeroAmount
		return
	}

	var feeBpsU, surchargeBpsU, net uint256.Int
	feeBpsU.SetUint64(c.tradeFeeBps)
	surchargeBpsU.SetUint64(c.surchargeBpsAt(now))
	if _, overflow := fee.MulOverflow(amountIn, &feeBpsU); overflow {
		err = ErrArithmetic
		return
	}
	fee.Div(&fee, bpsU)
	if _, overflow := surcharge.MulOverflow(amountIn, &surchargeBpsU); overflow {
		err = ErrArithmetic
		return
	}
	surcharge.Div(&surcharge, bpsU)
	net.Sub(amountIn, &fee)
	net.Sub(&net, &surcharge)
	if net.IsZero() {
		err = ErrZeroAmount
		return
	}

	// quoteToTokens: newToken = k / (vq + net); tokensOut = tokenReserve - newToken
	var vq, k, denominator, newToken uint256.Int
	if _, overflow := vq.AddOverflow(&c.virtualQuote, &c.quoteReserve); overflow {
		err = ErrArithmetic
		return
	}
	if _, overflow := k.MulOverflow(&vq, &c.tokenReserve); overflow {
		err = ErrArithmetic
		return
	}
	if _, overflow := denominator.AddOverflow(&vq, &net); overflow {
		err = ErrArithmetic
		return
	}
	newToken.Div(&k, &denominator)
	tokensOut.Sub(&c.tokenReserve, &newToken)
	if tokensOut.IsZero() {
		err = ErrZeroAmount
		return
	}
	return
}

// quoteSell prices a sell of tokensIn tokens. It returns the quote out and the fee, and the gross
// amount the curve gives up (out plus fee), clamped to the reserve the curve actually holds.
func (c *curveState) quoteSell(tokensIn *uint256.Int) (quoteOut, fee, gross uint256.Int, err error) {
	if tokensIn.IsZero() {
		err = ErrZeroAmount
		return
	}

	// tokensToQuote: newQuote = k / (tokenReserve + tokensIn); gross = vq - newQuote
	var vq, k, denominator, newQuote uint256.Int
	if _, overflow := vq.AddOverflow(&c.virtualQuote, &c.quoteReserve); overflow {
		err = ErrArithmetic
		return
	}
	if _, overflow := k.MulOverflow(&vq, &c.tokenReserve); overflow {
		err = ErrArithmetic
		return
	}
	if _, overflow := denominator.AddOverflow(&c.tokenReserve, tokensIn); overflow {
		err = ErrArithmetic
		return
	}
	newQuote.Div(&k, &denominator)
	gross.Sub(&vq, &newQuote)

	// Division truncates, so gross rounds up and can exceed the real reserve by dust on the
	// final exit. The launchpad clamps rather than reverts, and so does this.
	if gross.Gt(&c.quoteReserve) {
		gross.Set(&c.quoteReserve)
	}
	if gross.IsZero() {
		err = ErrZeroAmount
		return
	}

	var feeBpsU uint256.Int
	feeBpsU.SetUint64(c.tradeFeeBps)
	if _, overflow := fee.MulOverflow(&gross, &feeBpsU); overflow {
		err = ErrArithmetic
		return
	}
	fee.Div(&fee, bpsU)
	quoteOut.Sub(&gross, &fee)
	if quoteOut.IsZero() {
		err = ErrZeroAmount
		return
	}
	return
}
