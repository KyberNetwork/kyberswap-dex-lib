package pools

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"slices"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo-v3/quoting"
)

const invalidTickNumber int32 = math.MinInt32

type (
	BasePoolSwapState struct {
		SqrtRatio       *uint256.Int `json:"sqrtRatio"`
		Liquidity       *uint256.Int `json:"liquidity"`
		ActiveTickIndex int          `json:"activeTickIndex"`
	}

	BasePoolState struct {
		*BasePoolSwapState
		SortedTicks []Tick   `json:"sortedTicks"`
		TickBounds  [2]int32 `json:"tickBounds"`
		ActiveTick  int32    `json:"activeTick"`
	}

	BasePool struct {
		key *ConcentratedPoolKey
		*BasePoolState
	}

	TickRPC struct {
		Number         int32    `json:"number"`
		LiquidityDelta *big.Int `json:"liquidityDelta"`
	}

	Tick struct {
		Number         int32       `json:"number"`
		LiquidityDelta *int256.Int `json:"liquidityDelta"`
	}

	nextInitializedTick struct {
		*Tick
		index     int
		sqrtRatio *uint256.Int
	}
)

func (s *BasePoolState) CloneState() *BasePoolState {
	cloned := *s
	cloned.SortedTicks = slices.Clone(s.SortedTicks)
	return &cloned
}

func (s *BasePoolState) UpdateTick(updatedTickNumber int32, liquidityDelta *int256.Int, upper, forceInsert bool) {
	ticks := s.SortedTicks

	liquidityDelta = liquidityDelta.Clone()
	if upper {
		liquidityDelta.Neg(liquidityDelta)
	}

	nearestTickIndex := NearestInitializedTickIndex(ticks, updatedTickNumber)

	var (
		nearestTick       *Tick
		nearestTickNumber = invalidTickNumber
	)

	if nearestTickIndex != -1 {
		nearestTick = &ticks[nearestTickIndex]
		nearestTickNumber = nearestTick.Number
	}

	newTickReferenced := nearestTickNumber != updatedTickNumber || nearestTick == nil
	if newTickReferenced {
		if !forceInsert && nearestTickIndex == -1 {
			delta := ticks[0].LiquidityDelta
			delta.Add(delta, liquidityDelta)
		} else if !forceInsert && nearestTickIndex == len(ticks)-1 {
			delta := ticks[len(ticks)-1].LiquidityDelta
			delta.Add(delta, liquidityDelta)
		} else {
			insertIdx := nearestTickIndex + 1

			s.SortedTicks = slices.Insert(ticks, insertIdx, Tick{
				Number:         updatedTickNumber,
				LiquidityDelta: liquidityDelta,
			})

			if s.ActiveTick >= updatedTickNumber {
				s.ActiveTickIndex++
			}
		}
	} else {
		newDelta := nearestTick.LiquidityDelta.Add(nearestTick.LiquidityDelta, liquidityDelta)

		if newDelta.IsZero() && !slices.Contains(s.TickBounds[:], nearestTickNumber) {
			s.SortedTicks = slices.Delete(ticks, nearestTickIndex, nearestTickIndex+1)
			if s.ActiveTick >= updatedTickNumber {
				s.ActiveTickIndex--
			}
		}
	}
}

func (s *BasePoolState) AddLiquidityCutoffs() {
	var currentLiquidity uint256.Int
	belowActiveTick := true
	var activeTickIndex int

	// The liquidity added/removed by out-of-range initialized ticks (i.e. lower than the min checked tick number)
	var liquidityDeltaMin uint256.Int

	for i, tick := range s.SortedTicks {
		if belowActiveTick && s.ActiveTick < tick.Number {
			belowActiveTick = false

			activeTickIndex = i - 1

			liquidityDeltaMin.Sub(s.Liquidity, &currentLiquidity)

			// We now need to switch to tracking the liquidity that needs to be cut off at the max checked tick number, therefore reset to the actual liquidity
			currentLiquidity.Set(s.Liquidity)
		}

		currentLiquidity.Add(&currentLiquidity, (*uint256.Int)(tick.LiquidityDelta))
	}

	if belowActiveTick {
		activeTickIndex = len(s.SortedTicks) - 1
		liquidityDeltaMin.Sub(s.Liquidity, &currentLiquidity)
		currentLiquidity.Set(s.Liquidity)
	}

	s.ActiveTickIndex = activeTickIndex

	s.UpdateTick(s.TickBounds[0], (*int256.Int)(&liquidityDeltaMin), false, true)
	s.UpdateTick(s.TickBounds[1], (*int256.Int)(&currentLiquidity), true, true)
}

func (p *BasePool) GetKey() IPoolKey {
	return p.key
}

func (p *BasePool) GetState() any {
	return p.BasePoolState
}

func (p *BasePool) CloneState() any {
	cloned := *p
	cloned.key = p.key.CloneState()
	cloned.BasePoolState = p.BasePoolState.CloneState()
	return &cloned
}

func (p *BasePool) SetSwapState(state any) {
	p.BasePoolSwapState = state.(*BasePoolSwapState)
}

func (p *BasePool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	var liquidity uint256.Int
	sqrtRatio := p.SqrtRatio.Clone()
	liquidity.Set(p.Liquidity)
	activeTickIndex := p.ActiveTickIndex

	isIncreasing := ekubomath.IsPriceIncreasing(amount, isToken1)

	sqrtRatioLimit := lo.Ternary(isIncreasing, ekubomath.MaxSqrtRatio, ekubomath.MinSqrtRatio)

	var calculatedAmount, feesPaid, amountRemaining, tmp uint256.Int
	var initializedTicksCrossed uint32 = 0
	amountRemaining.Set(amount)

	startingSqrtRatio := p.SqrtRatio

	for !amountRemaining.IsZero() && !sqrtRatio.Eq(sqrtRatioLimit) {
		var nextInitTick *nextInitializedTick
		if isIncreasing {
			nextTickIndex := activeTickIndex + 1
			if nextTickIndex < len(p.SortedTicks) {
				tick := &p.SortedTicks[nextTickIndex]
				nextInitTick = &nextInitializedTick{
					Tick:      tick,
					index:     nextTickIndex,
					sqrtRatio: ekubomath.ToSqrtRatio(tick.Number),
				}
			}
		} else if activeTickIndex != -1 {
			tick := &p.SortedTicks[activeTickIndex]
			nextInitTick = &nextInitializedTick{
				Tick:      tick,
				index:     activeTickIndex,
				sqrtRatio: ekubomath.ToSqrtRatio(tick.Number),
			}
		}

		var stepSqrtRatioLimit *uint256.Int
		if nextInitTick == nil {
			stepSqrtRatioLimit = sqrtRatioLimit
		} else {
			nextRatio := nextInitTick.sqrtRatio
			stepSqrtRatioLimit = lo.Ternary(nextRatio.Lt(sqrtRatioLimit) == isIncreasing, nextRatio, sqrtRatioLimit)
		}

		step, err := ekubomath.ComputeStep(
			sqrtRatio,
			&liquidity,
			stepSqrtRatioLimit,
			&amountRemaining,
			isToken1,
			p.key.Config.Fee,
		)
		if err != nil {
			return nil, fmt.Errorf("swap step computation: %w", err)
		}

		amountRemaining.Sub(&amountRemaining, step.ConsumedAmount)
		calculatedAmount.Add(&calculatedAmount, step.CalculatedAmount)
		feesPaid.Add(&feesPaid, step.FeeAmount)
		sqrtRatio = step.SqrtRatioNext

		if nextInitTick != nil {
			tickIndex := nextInitTick.index
			if sqrtRatio.Eq(nextInitTick.sqrtRatio) {
				if isIncreasing {
					activeTickIndex = tickIndex
				} else {
					activeTickIndex = tickIndex - 1
				}

				initializedTicksCrossed += 1

				liquidityDelta := nextInitTick.LiquidityDelta
				liquidityDeltaAbs := tmp.Abs((*uint256.Int)(liquidityDelta))
				if (liquidityDelta.Sign() > 0) == isIncreasing {
					liquidity.Add(&liquidity, liquidityDeltaAbs)
				} else {
					liquidity.Sub(&liquidity, liquidityDeltaAbs)
				}
			}
		}
	}

	tickSpacingsCrossed := ekubomath.ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, sqrtRatio,
		p.key.Config.TypeConfig.TickSpacing)

	var skipAhead uint32
	if initializedTicksCrossed != 0 {
		skipAhead = tickSpacingsCrossed / initializedTicksCrossed
	}

	priceLimit := sqrtRatioLimit
	if isIncreasing {
		if upperTickBound := p.TickBounds[1] + 1; upperTickBound < ekubomath.MaxTick {
			priceLimit = ekubomath.ToSqrtRatio(upperTickBound)
		}
	} else if lowerTickBound := p.TickBounds[0] - 1; lowerTickBound > ekubomath.MinTick {
		priceLimit = ekubomath.ToSqrtRatio(lowerTickBound)
	}
	priceLimit = ekubomath.FixedSqrtRatioToFloat(priceLimit, isIncreasing)

	return &quoting.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, &amountRemaining),
		CalculatedAmount: &calculatedAmount,
		FeesPaid:         &feesPaid,
		Gas:              quoting.BaseGasConcentratedLiquiditySwap + int64(initializedTicksCrossed)*quoting.GasInitializedTickCrossed + int64(tickSpacingsCrossed)*quoting.GasTickSpacingCrossed,
		SwapInfo: quoting.SwapInfo{
			SkipAhead:  skipAhead,
			IsToken1:   isToken1,
			PriceLimit: priceLimit,
			SwapStateAfter: &BasePoolSwapState{
				sqrtRatio,
				&liquidity,
				activeTickIndex,
			},
			TickSpacingsCrossed: tickSpacingsCrossed,
		},
	}, nil
}

func NewBasePool(key *ConcentratedPoolKey, state *BasePoolState) *BasePool {
	return &BasePool{
		key:           key,
		BasePoolState: state,
	}
}

func NearestInitializedTickIndex(sortedTicks []Tick, tickNumber int32) int {
	idx, found := slices.BinarySearchFunc(sortedTicks, tickNumber, func(tick Tick, tickNumber int32) int {
		return cmp.Compare(tick.Number, tickNumber)
	})

	if !found {
		idx--
	}

	return idx
}
