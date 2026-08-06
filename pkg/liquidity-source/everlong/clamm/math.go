package everlongclamm

import (
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// swapExactInput is a wei-exact port of CLPool.swap for the exact-input, feePips=0 path:
// unlike the generic V3 walk in pkg/liquidity-source/uniswap/v3 (which jumps between
// initialized ticks and approximates bitmap-word rounding), it steps one bitmap word at
// a time exactly like nextInitializedTickWithinOneWord on-chain. On this pool the
// active liquidity is small enough that the approximation's wei-level input accounting
// shifts the output by ~5e-9 relatively — and can overshoot the chain; the per-word walk
// reproduces the on-chain output to the wei (validated against CLPoolManager at a pinned
// block).
type swapState struct {
	SqrtPriceX96       uint256.Int
	Tick               int
	Liquidity          uint256.Int
	AmountOut          uint256.Int
	RemainingAmountIn  uint256.Int
	CrossInitTickLoops int
}

func swapExactInput(p *uniswapv3.Pool, zeroForOne bool, amountIn,
	sqrtPriceLimitX96 *uint256.Int) (swapState, error) {
	state := swapState{
		SqrtPriceX96: p.SqrtRatioX96,
		Tick:         p.TickCurrent,
		Liquidity:    p.Liquidity,
	}
	state.RemainingAmountIn.Set(amountIn)

	var sqrtPriceNextX96, targetValue, nxtSqrtPriceX96, stepAmountIn, stepAmountOut, feeAmount uint256.Int
	for !state.RemainingAmountIn.IsZero() && !state.SqrtPriceX96.Eq(sqrtPriceLimitX96) {
		tickNext, initialized := nextTickWithinOneWord(p.Ticks, state.Tick, p.TickSpacing, zeroForOne)
		if tickNext < uniswapv3.MinTick {
			tickNext = uniswapv3.MinTick
		} else if tickNext > uniswapv3.MaxTick {
			tickNext = uniswapv3.MaxTick
		}
		if err := uniswapv3.GetSqrtRatioAtTick(tickNext, &sqrtPriceNextX96); err != nil {
			return state, err
		}

		if zeroForOne && sqrtPriceLimitX96.Gt(&sqrtPriceNextX96) ||
			!zeroForOne && sqrtPriceLimitX96.Lt(&sqrtPriceNextX96) {
			targetValue.Set(sqrtPriceLimitX96)
		} else {
			targetValue.Set(&sqrtPriceNextX96)
		}

		if err := uniswapv3.ComputeSwapStep(&state.SqrtPriceX96, &targetValue, &state.Liquidity,
			&state.RemainingAmountIn, 0, &nxtSqrtPriceX96, &stepAmountIn, &stepAmountOut,
			&feeAmount); err != nil {
			return state, err
		}
		state.SqrtPriceX96.Set(&nxtSqrtPriceX96)
		state.RemainingAmountIn.Sub(&state.RemainingAmountIn, stepAmountIn.Add(&stepAmountIn, &feeAmount))
		state.AmountOut.Add(&state.AmountOut, &stepAmountOut)

		if state.SqrtPriceX96.Eq(&sqrtPriceNextX96) {
			if initialized {
				tickData, err := getTick(p.Ticks, tickNext)
				if err != nil {
					return state, err
				}
				if zeroForOne {
					state.Liquidity.Sub(&state.Liquidity, (*uint256.Int)(tickData.LiquidityNet))
				} else {
					state.Liquidity.Add(&state.Liquidity, (*uint256.Int)(tickData.LiquidityNet))
				}
				state.CrossInitTickLoops++
			}
			if zeroForOne {
				state.Tick = tickNext - 1
			} else {
				state.Tick = tickNext
			}
		} else if !state.SqrtPriceX96.Eq(&targetValue) {
			// input exhausted mid-segment
			tick, err := uniswapv3.GetTickAtSqrtRatio(&state.SqrtPriceX96)
			if err != nil {
				return state, err
			}
			state.Tick = tick
		}
	}
	return state, nil
}

// nextTickWithinOneWord mirrors TickBitmap.nextInitializedTickWithinOneWord over the
// sorted tick slice: the next initialized tick in the current 256-slot bitmap word, or
// the word's edge tick (uninitialized) when the word has none further.
func nextTickWithinOneWord(ticks []uniswapv3.TickU256, tick, tickSpacing int, lte bool) (int, bool) {
	if lte {
		compressed := floorDiv(tick, tickSpacing)
		wordStart := compressed - mod256(compressed)
		// largest initialized tick t with wordStart*spacing <= t <= tick
		idx := largestAtOrBelow(ticks, tick)
		if idx >= 0 && ticks[idx].Index >= wordStart*tickSpacing {
			return ticks[idx].Index, true
		}
		return wordStart * tickSpacing, false
	}
	compressed := floorDiv(tick, tickSpacing) + 1
	wordEnd := compressed + (255 - mod256(compressed))
	// smallest initialized tick t with compressed*spacing <= t <= wordEnd*spacing
	idx := largestAtOrBelow(ticks, compressed*tickSpacing-1) + 1
	if idx < len(ticks) && ticks[idx].Index <= wordEnd*tickSpacing {
		return ticks[idx].Index, true
	}
	return wordEnd * tickSpacing, false
}

func getTick(ticks []uniswapv3.TickU256, index int) (uniswapv3.TickU256, error) {
	idx := largestAtOrBelow(ticks, index)
	if idx < 0 || ticks[idx].Index != index {
		return uniswapv3.TickU256{}, uniswapv3.ErrInvalidTickIndex
	}
	return ticks[idx], nil
}

// largestAtOrBelow returns the index of the largest tick with Index <= tick, or -1.
func largestAtOrBelow(ticks []uniswapv3.TickU256, tick int) int {
	lo, hi := 0, len(ticks)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if ticks[mid].Index <= tick {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return lo - 1
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func mod256(a int) int {
	return ((a % 256) + 256) % 256
}
