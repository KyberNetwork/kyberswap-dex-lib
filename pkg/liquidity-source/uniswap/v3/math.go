package uniswapv3

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	errFeeTooHigh               = errors.New("fee too high")
	errInvalidSqrtRatioX96      = errors.New("invalid sqrtRatioX96")
	errSqrtPriceLimitX96TooLow  = errors.New("SqrtPriceLimitX96 too low")
	errSqrtPriceLimitX96TooHigh = errors.New("SqrtPriceLimitX96 too high")
	errBelowSmallest            = errors.New("below smallest")
	errAtOrAboveLargest         = errors.New("at or above largest")
	errEmptyTickList            = errors.New("empty tick list")
	errInvalidTickIndex         = errors.New("invalid tick index")
	errZeroTickSpacing          = errors.New("tick spacing must be greater than 0")
	errInvalidTickSpacing       = errors.New("invalid tick spacing")
	errZeroNet                  = errors.New("tick net delta must be zero")
	errSorted                   = errors.New("ticks must be sorted")
)

// Pool holds the mutable state needed to simulate Uniswap V3 swaps.
type Pool struct {
	Fee          FeeAmount
	SqrtRatioX96 *uint256.Int
	Liquidity    *uint256.Int
	TickCurrent  int
	ticks        []TickU256
}

type swapResult struct {
	amountCalculated   *int256.Int
	sqrtRatioX96       *uint256.Int
	liquidity          *uint256.Int
	remainingAmountIn  *int256.Int
	currentTick        int
	crossInitTickLoops int
}

// GetAmountResultV2 is returned by getOutputAmountV2 and getInputAmountV2.
type GetAmountResultV2 struct {
	ReturnedAmount     *int256.Int
	RemainingAmountIn  *int256.Int
	SqrtRatioX96       *uint256.Int
	Liquidity          *uint256.Int
	CurrentTick        int
	CrossInitTickLoops int
}

// newPool constructs a Pool, validating the fee tier and sqrtRatioX96 range.
func newPool(fee FeeAmount, sqrtRatioX96 *uint256.Int, liquidity *uint256.Int, tickCurrent int, ticks []TickU256, tickSpacing int) (*Pool, error) {
	if fee >= FeeMax {
		return nil, errFeeTooHigh
	}
	if err := validateList(ticks, tickSpacing); err != nil {
		return nil, err
	}
	var lo, hi uint256.Int
	if err := GetSqrtRatioAtTick(tickCurrent, &lo); err != nil {
		return nil, err
	}
	if err := GetSqrtRatioAtTick(tickCurrent+1, &hi); err != nil {
		return nil, err
	}
	if sqrtRatioX96.Cmp(&lo) < 0 || sqrtRatioX96.Cmp(&hi) > 0 {
		return nil, errInvalidSqrtRatioX96
	}
	return &Pool{
		Fee:          fee,
		SqrtRatioX96: sqrtRatioX96,
		Liquidity:    liquidity,
		TickCurrent:  tickCurrent,
		ticks:        ticks,
	}, nil
}

// getOutputAmountV2 computes the output for an exact-input swap (positive amountSpecified).
func (p *Pool) getOutputAmountV2(inputAmount *int256.Int, zeroForOne bool, sqrtPriceLimitX96 *uint256.Int) (*GetAmountResultV2, error) {
	sr, err := p.swap(zeroForOne, inputAmount, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	return &GetAmountResultV2{
		ReturnedAmount:     new(int256.Int).Neg(sr.amountCalculated),
		RemainingAmountIn:  new(int256.Int).Set(sr.remainingAmountIn),
		SqrtRatioX96:       sr.sqrtRatioX96,
		Liquidity:          sr.liquidity,
		CurrentTick:        sr.currentTick,
		CrossInitTickLoops: sr.crossInitTickLoops,
	}, nil
}

// getInputAmountV2 computes the input for an exact-output swap.
// outputAmount must be positive. Returns (inputAmount, newPoolState, err).
func (p *Pool) getInputAmountV2(outputAmount *int256.Int, zeroForOne bool, sqrtPriceLimitX96 *uint256.Int) (*int256.Int, *Pool, error) {
	negOut := new(int256.Int).Neg(outputAmount)
	sr, err := p.swap(zeroForOne, negOut, sqrtPriceLimitX96)
	if err != nil {
		return nil, nil, err
	}
	newPool := &Pool{
		Fee:          p.Fee,
		SqrtRatioX96: sr.sqrtRatioX96,
		Liquidity:    sr.liquidity,
		TickCurrent:  sr.currentTick,
		ticks:        p.ticks,
	}
	return sr.amountCalculated, newPool, nil
}

// swap is the core Uniswap V3 swap algorithm.
// Positive amountSpecified → exact input. Negative → exact output.
func (p *Pool) swap(zeroForOne bool, amountSpecified *int256.Int, sqrtPriceLimitX96 *uint256.Int) (*swapResult, error) {
	if sqrtPriceLimitX96 == nil {
		if zeroForOne {
			sqrtPriceLimitX96 = new(uint256.Int).AddUint64(MinSqrtRatioU256, 1)
		} else {
			sqrtPriceLimitX96 = new(uint256.Int).SubUint64(MaxSqrtRatioU256, 1)
		}
	}

	if zeroForOne {
		if sqrtPriceLimitX96.Cmp(MinSqrtRatioU256) < 0 {
			return nil, errSqrtPriceLimitX96TooLow
		}
		if sqrtPriceLimitX96.Cmp(p.SqrtRatioX96) >= 0 {
			return nil, errSqrtPriceLimitX96TooHigh
		}
	} else {
		if sqrtPriceLimitX96.Cmp(MaxSqrtRatioU256) > 0 {
			return nil, errSqrtPriceLimitX96TooHigh
		}
		if sqrtPriceLimitX96.Cmp(p.SqrtRatioX96) <= 0 {
			return nil, errSqrtPriceLimitX96TooLow
		}
	}

	exactInput := amountSpecified.Sign() >= 0

	amountSpecifiedRemaining := new(int256.Int).Set(amountSpecified)
	amountCalculated := int256.NewInt(0)
	sqrtPriceX96 := new(uint256.Int).Set(p.SqrtRatioX96)
	tick := p.TickCurrent
	liquidity := new(uint256.Int).Set(p.Liquidity)
	crossInitTickLoops := 0

	for !amountSpecifiedRemaining.IsZero() && sqrtPriceX96.Cmp(sqrtPriceLimitX96) != 0 {
		var sqrtPriceStartX96, sqrtPriceNextX96 uint256.Int
		sqrtPriceStartX96.Set(sqrtPriceX96)

		tickNext, initialized, err := nextInitializedTickIndex(p.ticks, tick, zeroForOne)
		if err != nil {
			return nil, err
		}
		if tickNext < MinTick {
			tickNext = MinTick
		} else if tickNext > MaxTick {
			tickNext = MaxTick
		}

		if err = GetSqrtRatioAtTick(tickNext, &sqrtPriceNextX96); err != nil {
			return nil, err
		}

		var targetValue uint256.Int
		if zeroForOne {
			if sqrtPriceNextX96.Cmp(sqrtPriceLimitX96) < 0 {
				targetValue.Set(sqrtPriceLimitX96)
			} else {
				targetValue.Set(&sqrtPriceNextX96)
			}
		} else {
			if sqrtPriceNextX96.Cmp(sqrtPriceLimitX96) > 0 {
				targetValue.Set(sqrtPriceLimitX96)
			} else {
				targetValue.Set(&sqrtPriceNextX96)
			}
		}

		var nxtSqrtPriceX96, amountIn, amountOut, feeAmount uint256.Int
		if err = ComputeSwapStep(sqrtPriceX96, &targetValue, liquidity, amountSpecifiedRemaining,
			p.Fee, &nxtSqrtPriceX96, &amountIn, &amountOut, &feeAmount); err != nil {
			return nil, err
		}
		sqrtPriceX96.Set(&nxtSqrtPriceX96)

		var amountInPlusFee uint256.Int
		amountInPlusFee.Add(&amountIn, &feeAmount)

		var amountInPlusFeeSigned, amountOutSigned int256.Int
		if err = ToInt256(&amountInPlusFee, &amountInPlusFeeSigned); err != nil {
			return nil, err
		}
		if err = ToInt256(&amountOut, &amountOutSigned); err != nil {
			return nil, err
		}

		if exactInput {
			amountSpecifiedRemaining.Sub(amountSpecifiedRemaining, &amountInPlusFeeSigned)
			amountCalculated.Sub(amountCalculated, &amountOutSigned)
		} else {
			amountSpecifiedRemaining.Add(amountSpecifiedRemaining, &amountOutSigned)
			amountCalculated.Add(amountCalculated, &amountInPlusFeeSigned)
		}

		if sqrtPriceX96.Cmp(&sqrtPriceNextX96) == 0 {
			if initialized {
				t, err := getTick(p.ticks, tickNext)
				if err != nil {
					return nil, err
				}
				liquidityNet := t.LiquidityNet
				if zeroForOne {
					liquidityNet = new(int256.Int).Neg(liquidityNet)
				}
				if err = AddDeltaInPlace(liquidity, liquidityNet); err != nil {
					return nil, err
				}
				crossInitTickLoops++
			}
			if zeroForOne {
				tick = tickNext - 1
			} else {
				tick = tickNext
			}
		} else if sqrtPriceX96.Cmp(&sqrtPriceStartX96) != 0 {
			var err error
			tick, err = GetTickAtSqrtRatio(sqrtPriceX96)
			if err != nil {
				return nil, err
			}
		}
	}

	return &swapResult{
		amountCalculated:   amountCalculated,
		sqrtRatioX96:       sqrtPriceX96,
		liquidity:          liquidity,
		remainingAmountIn:  amountSpecifiedRemaining,
		currentTick:        tick,
		crossInitTickLoops: crossInitTickLoops,
	}, nil
}

// ---------- Tick list helpers ----------

func validateList(ticks []TickU256, tickSpacing int) error {
	if tickSpacing <= 0 {
		return errZeroTickSpacing
	}
	for _, t := range ticks {
		if t.Index%tickSpacing != 0 {
			return errInvalidTickSpacing
		}
	}
	sum := int256.NewInt(0)
	for _, t := range ticks {
		sum.Add(sum, t.LiquidityNet)
	}
	if !sum.IsZero() {
		return errZeroNet
	}
	for i := 0; i < len(ticks)-1; i++ {
		if ticks[i].Index > ticks[i+1].Index {
			return errSorted
		}
	}
	return nil
}

func isBelowSmallest(ticks []TickU256, tick int) (bool, error) {
	if len(ticks) == 0 {
		return true, errEmptyTickList
	}
	return tick < ticks[0].Index, nil
}

func isAtOrAboveLargest(ticks []TickU256, tick int) (bool, error) {
	if len(ticks) == 0 {
		return true, errEmptyTickList
	}
	return tick >= ticks[len(ticks)-1].Index, nil
}

func getTick(ticks []TickU256, index int) (TickU256, error) {
	idx, err := binarySearch(ticks, index)
	if err != nil {
		return TickU256{}, err
	}
	if idx < 0 || ticks[idx].Index != index {
		return TickU256{}, errInvalidTickIndex
	}
	return ticks[idx], nil
}

func nextInitializedTick(ticks []TickU256, tick int, lte bool) (TickU256, error) {
	if lte {
		below, err := isBelowSmallest(ticks, tick)
		if err != nil {
			return TickU256{}, err
		}
		if below {
			return TickU256{}, errBelowSmallest
		}
		above, err := isAtOrAboveLargest(ticks, tick)
		if err != nil {
			return TickU256{}, err
		}
		if above {
			return ticks[len(ticks)-1], nil
		}
		idx, err := binarySearch(ticks, tick)
		if err != nil {
			return TickU256{}, err
		}
		return ticks[idx], nil
	}
	above, err := isAtOrAboveLargest(ticks, tick)
	if err != nil {
		return TickU256{}, err
	}
	if above {
		return TickU256{}, errAtOrAboveLargest
	}
	below, err := isBelowSmallest(ticks, tick)
	if err != nil {
		return TickU256{}, err
	}
	if below {
		return ticks[0], nil
	}
	idx, err := binarySearch(ticks, tick)
	if err != nil {
		return TickU256{}, err
	}
	return ticks[idx+1], nil
}

func nextInitializedTickIndex(ticks []TickU256, tick int, lte bool) (int, bool, error) {
	t, err := nextInitializedTick(ticks, tick, lte)
	if err != nil {
		return 0, false, err
	}
	return t.Index, !t.LiquidityGross.IsZero(), nil
}

func binarySearch(ticks []TickU256, tick int) (int, error) {
	below, err := isBelowSmallest(ticks, tick)
	if err != nil {
		return 0, err
	}
	if below {
		return 0, errBelowSmallest
	}
	start, end := 0, len(ticks)-1
	for start <= end {
		mid := (start + end) / 2
		if ticks[mid].Index == tick {
			return mid, nil
		} else if ticks[mid].Index < tick {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	if start < len(ticks) && ticks[start].Index < tick {
		return start, nil
	}
	return start - 1, nil
}
