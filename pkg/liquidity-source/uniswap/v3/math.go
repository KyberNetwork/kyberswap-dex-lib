package uniswapv3

import (
	"errors"

	"github.com/KyberNetwork/kutils"
	"github.com/holiman/uint256"
)

var (
	ErrFeeTooHigh               = errors.New("fee too high")
	ErrInvalidSqrtRatioX96      = errors.New("invalid sqrtRatioX96")
	ErrSqrtPriceLimitX96TooLow  = errors.New("SqrtPriceLimitX96 too low")
	ErrSqrtPriceLimitX96TooHigh = errors.New("SqrtPriceLimitX96 too high")
	ErrBelowSmallest            = errors.New("below smallest")
	ErrAtOrAboveLargest         = errors.New("at or above largest")
	ErrEmptyTickList            = errors.New("empty tick list")
	ErrInvalidTickIndex         = errors.New("invalid tick index")
	ErrZeroTickSpacing          = errors.New("tick spacing must be greater than 0")
	ErrInvalidTickSpacing       = errors.New("invalid tick spacing")
	ErrZeroNet                  = errors.New("tick net delta must be zero")
	ErrSorted                   = errors.New("ticks must be sorted")
)

// Pool holds the mutable state needed to simulate Uniswap V3 swaps.
type Pool struct {
	Fee            FeeAmount
	TickSpacing    int
	TickCurrent    int
	SqrtRatioX96   uint256.Int
	Liquidity      uint256.Int
	Ticks          []TickU256
	TickSqrtPrices []uint256.Int // precomputed GetSqrtRatioAtTick(Ticks[i].Index); read-only after construction
}

type SwapResult struct {
	AmountCalculated   uint256.Int
	SqrtRatioX96       uint256.Int
	Liquidity          uint256.Int
	RemainingAmountIn  uint256.Int
	CurrentTick        int
	CrossInitTickLoops int
}

// NewPool constructs a Pool, validating the fee tier and sqrtRatioX96 range.
func NewPool(fee FeeAmount, sqrtRatioX96 uint256.Int, liquidity uint256.Int, tickCurrent int, ticks []TickU256,
	tickSpacing int) (*Pool, error) {
	if fee >= FeeMax {
		return nil, ErrFeeTooHigh
	} else if err := validateList(ticks, tickSpacing); err != nil {
		return nil, err
	}
	var sqrtP uint256.Int
	if err := GetSqrtRatioAtTick(tickCurrent, &sqrtP); err != nil {
		return nil, err
	} else if sqrtRatioX96.Lt(&sqrtP) {
		return nil, ErrInvalidSqrtRatioX96
	} else if err = GetSqrtRatioAtTick(tickCurrent+1, &sqrtP); err != nil {
		return nil, err
	} else if !sqrtRatioX96.Lt(&sqrtP) {
		return nil, ErrInvalidSqrtRatioX96
	}
	sqrtPrices := make([]uint256.Int, len(ticks))
	for i, t := range ticks {
		if err := GetSqrtRatioAtTick(t.Index, &sqrtPrices[i]); err != nil {
			return nil, err
		}
	}
	return &Pool{
		Fee:            fee,
		TickSpacing:    tickSpacing,
		TickCurrent:    tickCurrent,
		SqrtRatioX96:   sqrtRatioX96,
		Liquidity:      liquidity,
		Ticks:          ticks,
		TickSqrtPrices: sqrtPrices,
	}, nil
}

// GetOutputAmountV2 computes the output for an exact-input swap (positive amountSpecified).
// Returns (outputAmount, swapInfo, crossedTicks, err).
func (p *Pool) GetOutputAmountV2(zeroForOne bool, inputAmount, sqrtPriceLimitX96 uint256.Int) (SwapResult, error) {
	sr, err := p.Swap(zeroForOne, inputAmount, sqrtPriceLimitX96)
	if err != nil {
		return SwapResult{}, err
	}
	sr.AmountCalculated.Neg(&sr.AmountCalculated)
	return sr, nil
}

// GetInputAmountV2 computes the input for an exact-output swap.
// outputAmount must be positive. Returns (inputAmount, swapInfo, err).
func (p *Pool) GetInputAmountV2(zeroForOne bool, outputAmount, sqrtPriceLimitX96 uint256.Int) (SwapResult, error) {
	outputAmount.Neg(&outputAmount)
	sr, err := p.Swap(zeroForOne, outputAmount, sqrtPriceLimitX96)
	if err != nil {
		return SwapResult{}, err
	}
	return sr, nil
}

// Swap is the core Uniswap V3 swap algorithm.
// Positive amountSpecified → exact input. Negative → exact output.
func (p *Pool) Swap(zeroForOne bool, amountSpecified, sqrtPriceLimitX96 uint256.Int) (SwapResult, error) {
	if sqrtPriceLimitX96.IsZero() {
		if zeroForOne {
			sqrtPriceLimitX96 = *MinSqrtRatioU256P1
		} else {
			sqrtPriceLimitX96 = *MaxSqrtRatioU256M1
		}
	} else if zeroForOne {
		if sqrtPriceLimitX96.Lt(MinSqrtRatioU256) {
			return SwapResult{}, ErrSqrtPriceLimitX96TooLow
		} else if !sqrtPriceLimitX96.Lt(&p.SqrtRatioX96) {
			return SwapResult{}, ErrSqrtPriceLimitX96TooHigh
		}
	} else {
		if sqrtPriceLimitX96.Gt(MaxSqrtRatioU256) {
			return SwapResult{}, ErrSqrtPriceLimitX96TooHigh
		} else if !sqrtPriceLimitX96.Gt(&p.SqrtRatioX96) {
			return SwapResult{}, ErrSqrtPriceLimitX96TooLow
		}
	}

	if len(p.Ticks) == 0 {
		return SwapResult{
			RemainingAmountIn: amountSpecified,
			SqrtRatioX96:      p.SqrtRatioX96,
			Liquidity:         p.Liquidity,
			CurrentTick:       p.TickCurrent,
		}, nil
	}

	amountSpecifiedRemaining := amountSpecified
	sqrtPriceX96 := p.SqrtRatioX96
	tick := p.TickCurrent
	liquidity := p.Liquidity
	exactInput := amountSpecified.Sign() >= 0
	var amountCalculated uint256.Int
	var crossInitTickLoops int

	for !amountSpecifiedRemaining.IsZero() && !sqrtPriceLimitX96.Eq(&sqrtPriceX96) {
		sqrtPriceStartX96 := sqrtPriceX96
		var sqrtPriceNextX96 uint256.Int

		tickNext, slicePos, initialized, err := nextInitializedTickPos(p.Ticks, tick, zeroForOne)
		if err != nil {
			return SwapResult{}, err
		} else if tickNext < MinTick {
			tickNext = MinTick
			sqrtPriceNextX96 = *MinSqrtRatioU256
		} else if tickNext > MaxTick {
			tickNext = MaxTick
			sqrtPriceNextX96 = *MaxSqrtRatioU256P1
		} else {
			sqrtPriceNextX96 = p.TickSqrtPrices[slicePos]
		}

		var targetValue uint256.Int
		if zeroForOne && sqrtPriceLimitX96.Gt(&sqrtPriceNextX96) ||
			!zeroForOne && sqrtPriceLimitX96.Lt(&sqrtPriceNextX96) {
			targetValue = sqrtPriceLimitX96
		} else {
			targetValue = sqrtPriceNextX96
		}

		var nxtSqrtPriceX96, amountIn, amountOut, feeAmount uint256.Int
		if err = ComputeSwapStep(&sqrtPriceX96, &targetValue, &liquidity, &amountSpecifiedRemaining,
			p.Fee, &nxtSqrtPriceX96, &amountIn, &amountOut, &feeAmount); err != nil {
			return SwapResult{}, err
		}

		// per-tick rounding: exactIn floors amountOut, exactOut ceils amountIn.
		// The on-chain swap loop steps one bitmap word (256 compressed ticks) at a time via
		// nextInitializedTickWithinOneWord; each step rounds amountOut down by ≤1 unit.
		// wordCrossings ≈ number of bitmap words traversed = ceil(tick-spacings / 256).
		fullyCrossed := sqrtPriceX96.Set(&nxtSqrtPriceX96).Eq(&sqrtPriceNextX96)
		crossedBonus := 0
		if fullyCrossed {
			crossedBonus = 1
		}
		tickSpacingsCrossed := (kutils.Abs(tickNext-tick) - crossedBonus) / p.TickSpacing
		wordCrossings := sqrtPriceNextX96.SetUint64(uint64(max(1, (tickSpacingsCrossed+255)/256)))
		if exactInput {
			amountOut.SDiv(&amountOut, wordCrossings).Mul(&amountOut, wordCrossings)
		} else {
			amountIn.Add(&amountIn, wordCrossings).SubUint64(&amountIn, 1).SDiv(&amountIn, wordCrossings).Mul(&amountIn,
				wordCrossings)
		}

		amountInPlusFee := feeAmount.Add(&amountIn, &feeAmount)
		if exactInput {
			amountSpecifiedRemaining.Sub(&amountSpecifiedRemaining, amountInPlusFee)
			amountCalculated.Sub(&amountCalculated, &amountOut)
		} else {
			amountSpecifiedRemaining.Add(&amountSpecifiedRemaining, &amountOut)
			amountCalculated.Add(&amountCalculated, amountInPlusFee)
		}

		if fullyCrossed {
			if initialized {
				t, err := getTick(p.Ticks, tickNext)
				if err != nil {
					return SwapResult{}, err
				} else if zeroForOne {
					liquidity.Sub(&liquidity, (*uint256.Int)(t.LiquidityNet))
				} else {
					liquidity.Add(&liquidity, (*uint256.Int)(t.LiquidityNet))
				}
				if liquidity.Sign() < 0 {
					return SwapResult{}, errOverflowUint128
				}
				crossInitTickLoops++
			}
			if zeroForOne {
				tick = tickNext - 1
			} else {
				tick = tickNext
			}
		} else if !sqrtPriceX96.Eq(&sqrtPriceStartX96) {
			if tick, err = GetTickAtSqrtRatio(&sqrtPriceX96); err != nil {
				return SwapResult{}, err
			}
		}
	}

	return SwapResult{
		AmountCalculated:   amountCalculated,
		SqrtRatioX96:       sqrtPriceX96,
		Liquidity:          liquidity,
		RemainingAmountIn:  amountSpecifiedRemaining,
		CurrentTick:        tick,
		CrossInitTickLoops: crossInitTickLoops,
	}, nil
}

// ---------- Tick list helpers ----------

func validateList(ticks []TickU256, tickSpacing int) error {
	if tickSpacing <= 0 {
		return ErrZeroTickSpacing
	}
	var sum uint256.Int
	for i, t := range ticks {
		if i > 0 && ticks[i-1].Index > ticks[i].Index {
			return ErrSorted
		} else if t.Index%tickSpacing != 0 {
			return ErrInvalidTickSpacing
		}
		sum.Add(&sum, (*uint256.Int)(t.LiquidityNet))
	}
	if !sum.IsZero() {
		return ErrZeroNet
	}
	return nil
}

func isBelowSmallest(ticks []TickU256, tick int) (bool, error) {
	if len(ticks) == 0 {
		return true, ErrEmptyTickList
	}
	return tick < ticks[0].Index, nil
}

func isAtOrAboveLargest(ticks []TickU256, tick int) (bool, error) {
	if len(ticks) == 0 {
		return true, ErrEmptyTickList
	}
	return tick >= ticks[len(ticks)-1].Index, nil
}

func getTick(ticks []TickU256, index int) (TickU256, error) {
	idx, err := binarySearch(ticks, index)
	if err != nil {
		return TickU256{}, err
	}
	if idx < 0 || ticks[idx].Index != index {
		return TickU256{}, ErrInvalidTickIndex
	}
	return ticks[idx], nil
}

// nextInitializedTickPos returns the tick value, its slice index, initialized
// flag, and any error.  The slice index lets the caller look up the
// precomputed sqrtPrice in Pool.TickSqrtPrices without a second binary search.
func nextInitializedTickPos(ticks []TickU256, tick int, lte bool) (tickVal, slicePos int, initialized bool, err error) {
	if lte {
		below, berr := isBelowSmallest(ticks, tick)
		if berr != nil {
			return 0, 0, false, berr
		} else if below {
			return 0, 0, false, ErrBelowSmallest
		}
		last := len(ticks) - 1
		if above, _ := isAtOrAboveLargest(ticks, tick); above {
			t := &ticks[last]
			return t.Index, last, !t.LiquidityGross.IsZero(), nil
		}
		idx := binarySearchRaw(ticks, tick)
		t := &ticks[idx]
		return t.Index, idx, !t.LiquidityGross.IsZero(), nil
	}
	if above, aerr := isAtOrAboveLargest(ticks, tick); aerr != nil {
		return 0, 0, false, aerr
	} else if above {
		return 0, 0, false, ErrAtOrAboveLargest
	}
	if below, _ := isBelowSmallest(ticks, tick); below {
		t := &ticks[0]
		return t.Index, 0, !t.LiquidityGross.IsZero(), nil
	}
	idx := binarySearchRaw(ticks, tick) + 1
	t := &ticks[idx]
	return t.Index, idx, !t.LiquidityGross.IsZero(), nil
}

// nextInitializedTickIndex is kept for callers outside of Swap that don't need
// the slice position.
func nextInitializedTickIndex(ticks []TickU256, tick int, lte bool) (int, bool, error) {
	tickVal, _, initialized, err := nextInitializedTickPos(ticks, tick, lte)
	return tickVal, initialized, err
}

func binarySearch(ticks []TickU256, tick int) (int, error) {
	below, err := isBelowSmallest(ticks, tick)
	if err != nil {
		return 0, err
	} else if below {
		return 0, ErrBelowSmallest
	}
	return binarySearchRaw(ticks, tick), nil
}

// binarySearchRaw finds the index of the largest tick with Index <= tick.
// Caller must guarantee tick >= ticks[0].Index (isBelowSmallest is false).
func binarySearchRaw(ticks []TickU256, tick int) int {
	start, end := 0, len(ticks)-1
	for start <= end {
		mid := (start + end) / 2
		if ticks[mid].Index == tick {
			return mid
		} else if ticks[mid].Index < tick {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return start - 1
}
