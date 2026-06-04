package uniswapv3

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kutils"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	Fee          FeeAmount
	TickSpacing  int
	TickCurrent  int
	SqrtRatioX96 *uint256.Int
	Liquidity    *uint256.Int
	Ticks        []TickU256
}

type SwapResult struct {
	AmountCalculated   *int256.Int
	SqrtRatioX96       *uint256.Int
	Liquidity          *uint256.Int
	RemainingAmountIn  *int256.Int
	CurrentTick        int
	CrossInitTickLoops int
}

// GetAmountResultV2 is returned by GetOutputAmountV2 and GetInputAmountV2.
type GetAmountResultV2 struct {
	ReturnedAmount     *int256.Int
	RemainingAmountIn  *int256.Int
	SqrtRatioX96       *uint256.Int
	Liquidity          *uint256.Int
	CurrentTick        int
	CrossInitTickLoops int
}

// NewPool constructs a Pool, validating the fee tier and sqrtRatioX96 range.
func NewPool(fee FeeAmount, sqrtRatioX96 *uint256.Int, liquidity *uint256.Int, tickCurrent int, ticks []TickU256,
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
	return &Pool{
		Fee:          fee,
		TickSpacing:  tickSpacing,
		TickCurrent:  tickCurrent,
		SqrtRatioX96: sqrtRatioX96,
		Liquidity:    liquidity,
		Ticks:        ticks,
	}, nil
}

// GetOutputAmountV2 computes the output for an exact-input swap (positive amountSpecified).
func (p *Pool) GetOutputAmountV2(inputAmount *int256.Int, zeroForOne bool,
	sqrtPriceLimitX96 *uint256.Int) (*GetAmountResultV2, error) {
	sr, err := p.Swap(zeroForOne, inputAmount, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}
	return &GetAmountResultV2{
		ReturnedAmount:     new(int256.Int).Neg(sr.AmountCalculated),
		RemainingAmountIn:  new(int256.Int).Set(sr.RemainingAmountIn),
		SqrtRatioX96:       sr.SqrtRatioX96,
		Liquidity:          sr.Liquidity,
		CurrentTick:        sr.CurrentTick,
		CrossInitTickLoops: sr.CrossInitTickLoops,
	}, nil
}

// GetInputAmountV2 computes the input for an exact-output swap.
// outputAmount must be positive. Returns (inputAmount, newPoolState, err).
func (p *Pool) GetInputAmountV2(outputAmount *int256.Int, zeroForOne bool, sqrtPriceLimitX96 *uint256.Int) (*int256.Int,
	*Pool, error) {
	negOut := new(int256.Int).Neg(outputAmount)
	sr, err := p.Swap(zeroForOne, negOut, sqrtPriceLimitX96)
	if err != nil {
		return nil, nil, err
	}
	pool := &Pool{
		Fee:          p.Fee,
		SqrtRatioX96: sr.SqrtRatioX96,
		Liquidity:    sr.Liquidity,
		TickCurrent:  sr.CurrentTick,
		Ticks:        p.Ticks,
	}
	return sr.AmountCalculated, pool, nil
}

// Swap is the core Uniswap V3 swap algorithm.
// Positive amountSpecified → exact input. Negative → exact output.
func (p *Pool) Swap(zeroForOne bool, amountSpecified *int256.Int, sqrtPriceLimitX96 *uint256.Int) (*SwapResult, error) {
	if sqrtPriceLimitX96 == nil {
		if zeroForOne {
			sqrtPriceLimitX96 = new(uint256.Int).AddUint64(MinSqrtRatioU256, 1)
		} else {
			sqrtPriceLimitX96 = new(uint256.Int).SubUint64(MaxSqrtRatioU256, 1)
		}
	}

	if zeroForOne {
		if sqrtPriceLimitX96.Lt(MinSqrtRatioU256) {
			return nil, ErrSqrtPriceLimitX96TooLow
		} else if !sqrtPriceLimitX96.Lt(p.SqrtRatioX96) {
			return nil, ErrSqrtPriceLimitX96TooHigh
		}
	} else {
		if sqrtPriceLimitX96.Gt(MaxSqrtRatioU256) {
			return nil, ErrSqrtPriceLimitX96TooHigh
		} else if !sqrtPriceLimitX96.Gt(p.SqrtRatioX96) {
			return nil, ErrSqrtPriceLimitX96TooLow
		}
	}

	exactInput := amountSpecified.Sign() >= 0

	amountSpecifiedRemaining := new(int256.Int).Set(amountSpecified)
	amountCalculated := int256.NewInt(0)
	sqrtPriceX96 := new(uint256.Int).Set(p.SqrtRatioX96)
	tick := p.TickCurrent
	liquidity := new(uint256.Int).Set(p.Liquidity)
	crossInitTickLoops := 0

	for !amountSpecifiedRemaining.IsZero() && !sqrtPriceX96.Eq(sqrtPriceLimitX96) {
		var sqrtPriceStartX96, sqrtPriceNextX96 uint256.Int
		sqrtPriceStartX96.Set(sqrtPriceX96)

		tickNext, initialized, err := nextInitializedTickIndex(p.Ticks, tick, zeroForOne)
		if err != nil {
			return nil, err
		} else if tickNext < MinTick {
			tickNext = MinTick
		} else if tickNext > MaxTick {
			tickNext = MaxTick
		}

		if err = GetSqrtRatioAtTick(tickNext, &sqrtPriceNextX96); err != nil {
			return nil, err
		}

		var targetValue uint256.Int
		targetValue.Set(lo.Ternary(zeroForOne, big256.Max, big256.Min)(sqrtPriceLimitX96, &sqrtPriceNextX96))

		var nxtSqrtPriceX96, amountIn, amountOut, feeAmount uint256.Int
		if err = ComputeSwapStep(sqrtPriceX96, &targetValue, liquidity, amountSpecifiedRemaining,
			p.Fee, &nxtSqrtPriceX96, &amountIn, &amountOut, &feeAmount); err != nil {
			return nil, err
		}

		// per-tick rounding: exactIn floors amountOut, exactOut ceils amountIn.
		// The on-chain swap loop steps one bitmap word (256 compressed ticks) at a time via
		// nextInitializedTickWithinOneWord; each step rounds amountOut down by ≤1 unit.
		// wordCrossings ≈ number of bitmap words traversed = ceil(tick-spacings / 256).
		fullyCrossed := sqrtPriceX96.Set(&nxtSqrtPriceX96).Eq(&sqrtPriceNextX96)
		tickSpacingsCrossed := (kutils.Abs(tickNext-tick) - lo.Ternary(fullyCrossed, 1, 0)) / p.TickSpacing
		wordCrossings := targetValue.SetUint64(uint64(max(1, (tickSpacingsCrossed+255)/256)))
		if exactInput {
			amountOut.SDiv(&amountOut, wordCrossings).Mul(&amountOut, wordCrossings)
		} else {
			amountIn.Add(&amountIn, wordCrossings).SubUint64(&amountIn, 1).SDiv(&amountIn, wordCrossings).Mul(&amountIn, wordCrossings)
		}

		amountInPlusFee := feeAmount.Add(&amountIn, &feeAmount)
		if exactInput {
			amountSpecifiedRemaining.Sub(amountSpecifiedRemaining, (*int256.Int)(amountInPlusFee))
			amountCalculated.Sub(amountCalculated, (*int256.Int)(&amountOut))
		} else {
			amountSpecifiedRemaining.Add(amountSpecifiedRemaining, (*int256.Int)(&amountOut))
			amountCalculated.Add(amountCalculated, (*int256.Int)(amountInPlusFee))
		}

		if fullyCrossed {
			if initialized {
				t, err := getTick(p.Ticks, tickNext)
				if err != nil {
					return nil, err
				}
				if lo.Ternary(zeroForOne, liquidity.Sub, liquidity.Add)(
					liquidity, (*uint256.Int)(t.LiquidityNet)).Sign() < 0 {
					return nil, errOverflowUint128
				}
				crossInitTickLoops++
			}
			tick = lo.Ternary(zeroForOne, tickNext-1, tickNext)
		} else if !sqrtPriceX96.Eq(&sqrtPriceStartX96) {
			if tick, err = GetTickAtSqrtRatio(sqrtPriceX96); err != nil {
				return nil, err
			}
		}
	}

	return &SwapResult{
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
	for _, t := range ticks {
		if t.Index%tickSpacing != 0 {
			return ErrInvalidTickSpacing
		}
	}
	var sum int256.Int
	for _, t := range ticks {
		sum.Add(&sum, t.LiquidityNet)
	}
	if !sum.IsZero() {
		return ErrZeroNet
	}
	for i := 0; i < len(ticks)-1; i++ {
		if ticks[i].Index > ticks[i+1].Index {
			return ErrSorted
		}
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

func nextInitializedTick(ticks []TickU256, tick int, lte bool) (TickU256, error) {
	if lte {
		below, err := isBelowSmallest(ticks, tick)
		if err != nil {
			return TickU256{}, err
		} else if below {
			return TickU256{}, ErrBelowSmallest
		}
		above, err := isAtOrAboveLargest(ticks, tick)
		if err != nil {
			return TickU256{}, err
		} else if above {
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
	} else if above {
		return TickU256{}, ErrAtOrAboveLargest
	}
	below, err := isBelowSmallest(ticks, tick)
	if err != nil {
		return TickU256{}, err
	} else if below {
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
	} else if below {
		return 0, ErrBelowSmallest
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
