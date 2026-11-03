package ephemeral

import (
	"context"
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

// The widest word range of a tick bitmap, that of a table indexed by tick. A bitmap lens narrows it
// to the words the pool's spacing can populate.
const (
	WordLower int16 = -3466
	WordUpper int16 = 3465
)

// TickSnapshot is what a bitmap lens's getSnapshot returns, one field per output in the ABI's order.
// V3Lens, AlgebraV1Lens, PancakeInfinityCLLens and FluidDexV2Lens all return it. V4Lens returns it
// without tickSpacing, which the pool key holds, and decodes with TickSpacing nil.
type TickSnapshot struct {
	TickSpacing  *big.Int
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Liquidity    *big.Int
	BlockNumber  *big.Int
	NextWordPos  int16
	Done         bool
	Ticks        []struct {
		Tick           *big.Int
		LiquidityNet   *big.Int
		LiquidityGross *big.Int
	}
}

// TickPool is a pool's state and every initialized tick, read through a bitmap lens at one block.
type TickPool struct {
	Block        uint64
	TickSpacing  *big.Int
	SqrtPriceX96 *big.Int
	Tick         int
	Liquidity    *big.Int
	Ticks        []Tick
}

// Check is CheckTicks of the pool's ticks against its liquidity.
func (p *TickPool) Check() error {
	return CheckTicks(p.Ticks, p.Tick, p.Liquidity)
}

// Node is the node a lens read pins its block on and calls: *ethclient.Client is one.
type Node interface {
	Chain
	Caller
}

// ReadTicks reads a pool through a bitmap lens at block (nil for the latest): the words [lower,
// upper], every page at one block and up to WordsParallel at once, the first through the Batcher
// batch asks for (see CallPage). args are the lens's arguments before the word range.
//
// A tick without liquidityGross is left out: some forks (monday-v3) leave its bitmap bit set once
// its liquidity is gone, and a swap would only pay to cross it.
func ReadTicks(ctx context.Context, c Node, batch *BatchConfig, contract *Contract, gas uint64,
	lower, upper int16, block *big.Int, args ...any) (*TickPool, error) {
	pin, err := Pin(ctx, c, block)
	if err != nil {
		return nil, err
	}
	var p TickPool
	read, ticks, err := ScanWords(ctx, lower, upper, pin, WordsParallel,
		func(ctx context.Context, from, to int16, block *big.Int) (Page[int16], []Tick, error) {
			var out TickSnapshot
			if err := CallPage(ctx, batch, c, from, lower, contract, &out, gas, block,
				append(args[:len(args):len(args)], from, to)...); err != nil {
				return Page[int16]{}, nil, err
			}
			if p.Liquidity == nil { // the first page, read before any other
				p.TickSpacing, p.SqrtPriceX96 = out.TickSpacing, out.SqrtPriceX96
				p.Tick, p.Liquidity = int(out.Tick.Int64()), out.Liquidity
			}
			var got []Tick
			for _, t := range out.Ticks {
				if t.LiquidityGross.Sign() == 0 {
					continue
				}
				got = append(got, Tick{Index: int(t.Tick.Int64()), LiquidityNet: t.LiquidityNet,
					LiquidityGross: t.LiquidityGross})
			}
			return Page[int16]{Block: out.BlockNumber.Uint64(), Next: out.NextWordPos, Done: out.Done}, got, nil
		})
	if err != nil {
		return nil, err
	}
	p.Block, p.Ticks = read, ticks
	return &p, nil
}
