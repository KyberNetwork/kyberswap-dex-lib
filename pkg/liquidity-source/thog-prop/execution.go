package thogprop

import "github.com/holiman/uint256"

// Hundredths of a basis point. Difference comparisons preserve strict fractional
// thresholds without multiplying (and potentially overflowing) the base fee.
func (ctx ExecutionContext) friction(fresh bool) uint64 {
	var friction uint64
	if ctx.AuctionHandlerWarm {
		friction = 1200
	} else if ctx.GasPrice.Gt(&ctx.BaseFee) {
		var priority, half, extra uint256.Int
		priority.Sub(&ctx.GasPrice, &ctx.BaseFee)
		half.Rsh(&ctx.BaseFee, 1)
		if priority.Gt(&ctx.BaseFee) {
			extra.Sub(&priority, &ctx.BaseFee)
			if extra.Gt(&half) {
				friction = 1200
			} else {
				friction = 800
			}
		} else if priority.Gt(&half) {
			friction = 600
		}
	}
	if fresh && friction != 0 {
		friction = friction/4 - 100
	}
	return friction
}

func applyExecutionFriction(m *arithmetic, amount uint256.Int, friction uint64) uint256.Int {
	return m.md(amount, m.sub(u(1_000_000), u(friction)), u(1_000_000), false)
}
