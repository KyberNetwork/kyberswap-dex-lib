package uniswapv3

import (
	"errors"

	"github.com/holiman/uint256"
)

// noSlicePos marks a step whose target tick did not come from Pool.Ticks, so it has no precomputed
// sqrt price in Pool.TickSqrtPrices.
const noSlicePos = -1

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

	// CrossEmptyWordLoops counts the swap-loop iterations that stopped at a tick-bitmap word edge
	// rather than at an initialized tick. On-chain each of those still pays for a bitmap word load
	// and a round of price math, so a swap can be expensive while crossing no initialized tick at
	// all — which is the whole shape of a swap through a thin pool.
	CrossEmptyWordLoops int
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

// GetInputAmountExactInEnd is GetInputAmountV2, and also where the exact-in swap of the input it
// returns leaves the pool, which is how a swap is executed. The two walks take the same input at
// every step but the last, where the exact-out walk rounds the price toward the pool and the
// exact-in one toward the swapper, so only that step is redone; the whole exact-in swap runs only
// when the redone step reaches its target and goes on. Over words priced in float the exact-out
// walk's error carries into the end it gives, as it does into its input.
func (p *Pool) GetInputAmountExactInEnd(zeroForOne bool, outputAmount, sqrtPriceLimitX96 uint256.Int) (
	SwapResult, SwapResult, error) {
	var last lastInputStep
	outputAmount.Neg(&outputAmount)
	sr, err := p.swap(zeroForOne, outputAmount, sqrtPriceLimitX96, &last)
	if err != nil {
		return SwapResult{}, SwapResult{}, err
	}
	end, ok, err := last.exactIn(p.Fee)
	if err == nil && !ok {
		end, err = p.Swap(zeroForOne, sr.AmountCalculated, sqrtPriceLimitX96)
	}
	if err != nil {
		return SwapResult{}, SwapResult{}, err
	}
	return sr, end, nil
}

// lastInputStep is the last step of an exact-out swap that took input: where it began, its target,
// the input it took, fee included, and the state it left.
type lastInputStep struct {
	start, target, liquidity, amountIn uint256.Int
	tick                               int
	reached, ok                        bool
	afterPrice, afterLiquidity         uint256.Int
	afterTick                          int
}

// exactIn redoes the step exact in with the input it took. Ending inside the step, it spends all of
// that input there, so its end is the exact-in swap's. Reaching the target as the exact-out step did,
// with the same input, it leaves the state that step left. Otherwise ok is false.
//
// The price the input less the fee moves to falls short of the target exactly when that input is
// less than the input to the target, as ComputeSwapStep decides it: each rounds the same way. So the
// step's amounts are worked out only when it reaches the target.
func (s *lastInputStep) exactIn(fee FeeAmount) (SwapResult, bool, error) {
	if !s.ok {
		return SwapResult{}, false, nil
	}
	zeroForOne := !s.start.Lt(&s.target)
	var lessFee, next uint256.Int
	_, err := mulDivSmall(&lessFee, &s.amountIn, maxFeeInt-uint64(fee), maxFeeInt)
	if err == nil {
		// past the price range it fails where the step reaches the target, which ComputeSwapStep takes
		err = GetNextSqrtPriceFromInput(&s.start, &s.liquidity, &lessFee, zeroForOne, &next)
	}
	if err == nil && (zeroForOne && next.Gt(&s.target) || !zeroForOne && next.Lt(&s.target)) {
		tick := s.tick
		if !next.Eq(&s.start) {
			if tick, err = GetTickAtSqrtRatio(&next); err != nil {
				return SwapResult{}, false, err
			}
		}
		return SwapResult{SqrtRatioX96: next, Liquidity: s.liquidity, CurrentTick: tick}, true, nil
	}
	var in, out, feeAmount uint256.Int
	if err := ComputeSwapStep(&s.start, &s.target, &s.liquidity, &s.amountIn, fee, &next, &in, &out,
		&feeAmount); err != nil {
		return SwapResult{}, false, err
	}
	if s.reached && feeAmount.Add(&in, &feeAmount).Eq(&s.amountIn) {
		return SwapResult{SqrtRatioX96: s.afterPrice, Liquidity: s.afterLiquidity, CurrentTick: s.afterTick}, true, nil
	}
	return SwapResult{}, false, nil
}

// Swap is the core Uniswap V3 swap algorithm.
// Positive amountSpecified → exact input. Negative → exact output.
func (p *Pool) Swap(zeroForOne bool, amountSpecified, sqrtPriceLimitX96 uint256.Int) (SwapResult, error) {
	return p.swap(zeroForOne, amountSpecified, sqrtPriceLimitX96, nil)
}

// swap is Swap, recording the last step that took input into last when last is not nil.
func (p *Pool) swap(zeroForOne bool, amountSpecified, sqrtPriceLimitX96 uint256.Int,
	last *lastInputStep) (SwapResult, error) {
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
	var crossInitTickLoops, crossEmptyWordLoops int
	paidWords := 0 // empty words walked exactly at nonzero liquidity, toward floatWalkAfterWords
	var upEdges *upEdgeTable
	// The next tick in the list stays the same while the loop walks empty words toward it.
	listTick, listPos, listInit, haveListTick := 0, 0, false, false

	for !amountSpecifiedRemaining.IsZero() && !sqrtPriceLimitX96.Eq(&sqrtPriceX96) {
		sqrtPriceStartX96 := sqrtPriceX96
		var sqrtPriceNextX96 uint256.Int

		var err error
		if !haveListTick {
			if listTick, listPos, listInit, err = nextInitializedTickPos(p.Ticks, tick, zeroForOne); err != nil {
				return SwapResult{}, err
			}
		}
		tickNext, slicePos, initialized := clampToWord(tick, p.TickSpacing, zeroForOne, listTick, listPos, listInit)
		haveListTick = slicePos == noSlicePos

		if tickNext < MinTick {
			tickNext = MinTick
			sqrtPriceNextX96 = *MinSqrtRatioU256
		} else if tickNext > MaxTick {
			tickNext = MaxTick
			sqrtPriceNextX96 = *MaxSqrtRatioU256P1
		} else if slicePos == noSlicePos && zeroForOne {
			sqrtPriceNextX96 = downEdgeSqrtPrice(tickNext)
		} else if slicePos == noSlicePos {
			if upEdges == nil {
				upEdges = upEdgeTableFor(p.TickSpacing)
			}
			if upEdges != nil {
				sqrtPriceNextX96 = upEdges.sqrtPrice(tickNext, p.TickSpacing)
			} else if err = GetSqrtRatioAtTick(tickNext, &sqrtPriceNextX96); err != nil {
				return SwapResult{}, err
			}
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
		if liquidity.IsZero() {
			// Every amount of the step is zero, so nothing rounds: the price moves straight to the target.
			nxtSqrtPriceX96 = targetValue
		} else if err = ComputeSwapStep(&sqrtPriceX96, &targetValue, &liquidity, &amountSpecifiedRemaining,
			p.Fee, &nxtSqrtPriceX96, &amountIn, &amountOut, &feeAmount); err != nil {
			return SwapResult{}, err
		}

		record := last != nil && (!amountIn.IsZero() || !feeAmount.IsZero())
		if record {
			last.start, last.target, last.liquidity, last.tick = sqrtPriceStartX96, targetValue, liquidity, tick
			last.amountIn.Add(&amountIn, &feeAmount)
			last.reached, last.ok = nxtSqrtPriceX96.Eq(&targetValue), true
		}

		// A step now spans at most one bitmap word, so ComputeSwapStep's own rounding is already
		// the on-chain rounding and needs no correction on top of it.
		fullyCrossed := sqrtPriceX96.Set(&nxtSqrtPriceX96).Eq(&sqrtPriceNextX96)

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
			} else {
				crossEmptyWordLoops++
				if !liquidity.IsZero() {
					paidWords++
				}
			}
			if zeroForOne {
				tick = tickNext - 1
			} else {
				tick = tickNext
			}
			if record {
				last.afterPrice, last.afterLiquidity, last.afterTick = sqrtPriceX96, liquidity, tick
			}
			if slicePos == noSlicePos && (liquidity.IsZero() || paidWords >= floatWalkAfterWords) {
				var words int
				if tick, words, err = p.walkWordsFloat(zeroForOne, exactInput, tick, listTick, &liquidity,
					&sqrtPriceLimitX96, &sqrtPriceX96, &amountSpecifiedRemaining, &amountCalculated); err != nil {
					return SwapResult{}, err
				}
				crossEmptyWordLoops += words
				if last != nil && words > 0 && !liquidity.IsZero() {
					last.ok = false // the words walked took input, priced in float
				}
			}
		} else if !sqrtPriceX96.Eq(&sqrtPriceStartX96) {
			if tick, err = GetTickAtSqrtRatio(&sqrtPriceX96); err != nil {
				return SwapResult{}, err
			}
		}
		if record && !fullyCrossed {
			last.afterPrice, last.afterLiquidity, last.afterTick = sqrtPriceX96, liquidity, tick
		}
	}

	return SwapResult{
		AmountCalculated:    amountCalculated,
		SqrtRatioX96:        sqrtPriceX96,
		Liquidity:           liquidity,
		RemainingAmountIn:   amountSpecifiedRemaining,
		CurrentTick:         tick,
		CrossInitTickLoops:  crossInitTickLoops,
		CrossEmptyWordLoops: crossEmptyWordLoops,
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

// nextInitializedTickWithinOneWord returns the tick the swap loop stops at next, mirroring
// TickBitmap.nextInitializedTickWithinOneWord: it searches only the bitmap word holding tick, so a
// run of empty words is walked one word per iteration rather than jumped in a single step.
//
// Reproducing that boundary is not an optimisation detail. Every stop rounds amountIn up to a whole
// unit and charges a fee on top of it, so in a pool thin enough that one word of price movement
// costs less than one unit of input, the per-word rounding is what the swap spends almost all of
// its input on. Collapsing the run into one step lets the input buy price movement it cannot pay
// for.
//
// slicePos indexes ticks when the stop is an initialized tick, and is noSlicePos when it is a word
// boundary — a boundary has no precomputed entry in Pool.TickSqrtPrices.
//
// This deliberately stops short of the contract in one place. Where the bitmap simply holds no bits
// past the ends of the tick list, and the SDK answers with the word edge, ErrBelowSmallest and
// ErrAtOrAboveLargest are propagated instead: reaching them means the tick list does not cover the
// current tick, which in this library usually means the tracker's tick set is incomplete rather
// than that the pool really has no liquidity there. Walking on at an assumed constant liquidity
// would turn missing data into a confident wrong quote.
func nextInitializedTickWithinOneWord(ticks []TickU256, tick, tickSpacing int, lte bool) (
	tickNext, slicePos int, initialized bool, err error) {
	tickVal, pos, init, err := nextInitializedTickPos(ticks, tick, lte)
	if err != nil {
		return 0, 0, false, err
	}
	tickNext, slicePos, initialized = clampToWord(tick, tickSpacing, lte, tickVal, pos, init)
	return tickNext, slicePos, initialized, nil
}

// clampToWord stops at the word edge when the next tick in the list (tickVal, pos, init) sits in a
// further word, which the bitmap search cannot see.
func clampToWord(tick, tickSpacing int, lte bool, tickVal, pos int, init bool) (int, int, bool) {
	if boundary := wordBoundaryTick(tick, tickSpacing, lte); (lte && boundary > tickVal) ||
		(!lte && boundary < tickVal) {
		return boundary, noSlicePos, false
	}
	return tickVal, pos, init
}

// wordBoundaryTick returns the tick that TickBitmap.nextInitializedTickWithinOneWord stops at when
// the bitmap word holding tick carries no initialized tick: the lowest tick of that word when
// searching down, the highest when searching up.
//
// The result may fall outside [MinTick, MaxTick] — a word straddles the bounds at both ends of the
// range — and the caller clamps it, exactly as the on-chain swap loop does.
func wordBoundaryTick(tick, tickSpacing int, zeroForOne bool) int {
	compressed := floorDiv(tick, tickSpacing)
	if !zeroForOne {
		// searching up starts at the tick after the current one
		compressed++
	}
	// >> is arithmetic in Go, so this rounds toward -inf just like Solidity's int24 >> 8.
	wordFirst := compressed >> 8 << 8
	if zeroForOne {
		return wordFirst * tickSpacing
	}
	return (wordFirst + 255) * tickSpacing
}

// floorDiv divides rounding toward negative infinity, which is what tick compression does on-chain
// and what Go's / does not.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
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
