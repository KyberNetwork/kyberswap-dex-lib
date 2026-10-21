package ephemeral

import (
	"errors"
	"fmt"
	"math/big"
)

// Tick is an initialized tick as tick-based contracts return it.
type Tick struct {
	Index          int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
}

// ErrTicksInconsistent means a tick set does not add up to the pool's own liquidity, so it lacks a
// tick or carries a wrong one.
var ErrTicksInconsistent = errors.New("ephemeral: ticks do not add up to the pool's liquidity")

// CheckTicks reports whether ticks, the pool's whole set read at one block, agree with its
// in-range liquidity at tick: liquidityNet sums to liquidity over the ticks at or below tick, and
// to zero over all of them. The first catches a position missing both its ticks while it straddles
// the price, the second a position missing one of them.
func CheckTicks(ticks []Tick, tick int, liquidity *big.Int) error {
	active, all := new(big.Int), new(big.Int)
	for _, t := range ticks {
		all.Add(all, t.LiquidityNet)
		if t.Index <= tick {
			active.Add(active, t.LiquidityNet)
		}
	}
	if all.Sign() != 0 {
		return fmt.Errorf("%w: liquidityNet sums to %s over all %d ticks", ErrTicksInconsistent, all, len(ticks))
	}
	if active.Cmp(liquidity) != 0 {
		return fmt.Errorf("%w: %s in range at tick %d, pool reports %s", ErrTicksInconsistent, active, tick,
			liquidity)
	}
	return nil
}
