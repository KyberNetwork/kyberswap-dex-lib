package pools

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"slices"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type BasePoolSwapState struct {
	SqrtRatio       *big.Int `json:"sqrtRatio"`
	Liquidity       *big.Int `json:"liquidity"`
	ActiveTickIndex int      `json:"activeTickIndex"`
}

type BasePoolState struct {
	*BasePoolSwapState
	SortedTicks []Tick   `json:"sortedTicks"`
	TickBounds  [2]int32 `json:"tickBounds"`
	ActiveTick  int32    `json:"activeTick"`
}

type BasePool struct {
	key *PoolKey
	*BasePoolState
}

type Tick struct {
	Number         int32    `json:"number"`
	LiquidityDelta *big.Int `json:"liquidityDelta"`
}

const invalidTickNumber int32 = math.MinInt32

func NearestInitializedTickIndex(sortedTicks []Tick, tickNumber int32) int {
	idx, found := slices.BinarySearchFunc(sortedTicks, tickNumber, func(tick Tick, tickNumber int32) int {
		return cmp.Compare(tick.Number, tickNumber)
	})

	if !found {
		idx--
	}

	return idx
}

func (s *BasePoolState) UpdateTick(updatedTickNumber int32, liquidityDelta *big.Int, upper, forceInsert bool) {
	ticks := s.SortedTicks

	liquidityDelta = new(big.Int).Set(liquidityDelta)
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

	newTickReferenced := nearestTickNumber != updatedTickNumber

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
		newDelta := new(big.Int).Add(nearestTick.LiquidityDelta, liquidityDelta)

		if newDelta.Sign() == 0 && !slices.Contains(s.TickBounds[:], nearestTickNumber) {
			s.SortedTicks = slices.Delete(ticks, nearestTickIndex, nearestTickIndex+1)

			if s.ActiveTick >= updatedTickNumber {
				s.ActiveTickIndex--
			}
		} else {
			nearestTick.LiquidityDelta = newDelta
		}
	}
}

func (s *BasePoolState) AddLiquidityCutoffs() {
	currentLiquidity := new(big.Int)
	belowActiveTick := true
	var activeTickIndex int

	// The liquidity added/removed by out-of-range initialized ticks (i.e. lower than the min checked tick number)
	liquidityDeltaMin := new(big.Int)

	for i, tick := range s.SortedTicks {
		if belowActiveTick && s.ActiveTick < tick.Number {
			belowActiveTick = false

			activeTickIndex = i - 1

			liquidityDeltaMin.Sub(s.Liquidity, currentLiquidity)

			// We now need to switch to tracking the liquidity that needs to be cut off at the max checked tick number, therefore reset to the actual liquidity
			currentLiquidity.Set(s.Liquidity)
		}

		currentLiquidity.Add(currentLiquidity, tick.LiquidityDelta)
	}

	if belowActiveTick {
		activeTickIndex = len(s.SortedTicks) - 1
		liquidityDeltaMin.Sub(s.Liquidity, currentLiquidity)
		currentLiquidity.Set(s.Liquidity)
	}

	s.ActiveTickIndex = activeTickIndex

	s.UpdateTick(s.TickBounds[0], liquidityDeltaMin, false, true)
	s.UpdateTick(s.TickBounds[1], currentLiquidity, true, true)
}

func NewBasePool(key *PoolKey, state *BasePoolState) *BasePool {
	return &BasePool{
		key,
		state,
	}
}

func (p *BasePool) GetKey() *PoolKey {
	return p.key
}

func (p *BasePool) GetState() any {
	return p.BasePoolState
}

func (p *BasePool) SetSwapState(state any) {
	p.BasePoolSwapState = state.(*BasePoolSwapState)
}

type nextInitializedTick struct {
	*Tick
	index     int
	sqrtRatio *big.Int
}

func (p *BasePool) Quote(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	sqrtRatio := new(big.Int).Set(p.SqrtRatio)
	liquidity := new(big.Int).Set(p.Liquidity)
	activeTickIndex := p.ActiveTickIndex

	isIncreasing := ekubomath.IsPriceIncreasing(amount, isToken1)

	var sqrtRatioLimit *big.Int
	if isIncreasing {
		sqrtRatioLimit = ekubomath.MaxSqrtRatio
	} else {
		sqrtRatioLimit = ekubomath.MinSqrtRatio
	}

	calculatedAmount := new(big.Int)
	feesPaid := new(big.Int)
	var initializedTicksCrossed uint32 = 0
	amountRemaining := new(big.Int).Set(amount)

	startingSqrtRatio := p.SqrtRatio

	for amountRemaining.Sign() != 0 && sqrtRatio.Cmp(sqrtRatioLimit) != 0 {
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

		var stepSqrtRatioLimit *big.Int
		if nextInitTick == nil {
			stepSqrtRatioLimit = sqrtRatioLimit
		} else {
			nextRatio := nextInitTick.sqrtRatio
			if (nextRatio.Cmp(sqrtRatioLimit) == -1) == isIncreasing {
				stepSqrtRatioLimit = nextRatio
			} else {
				stepSqrtRatioLimit = sqrtRatioLimit
			}
		}

		step, err := ekubomath.ComputeStep(
			sqrtRatio,
			liquidity,
			stepSqrtRatioLimit,
			amountRemaining,
			isToken1,
			p.key.Config.Fee,
		)
		if err != nil {
			return nil, fmt.Errorf("swap step computation: %w", err)
		}

		amountRemaining.Sub(amountRemaining, step.ConsumedAmount)
		calculatedAmount.Add(calculatedAmount, step.CalculatedAmount)
		feesPaid.Add(feesPaid, step.FeeAmount)
		sqrtRatio = step.SqrtRatioNext

		if nextInitTick != nil {
			tickIndex := nextInitTick.index
			if sqrtRatio.Cmp(nextInitTick.sqrtRatio) == 0 {
				if isIncreasing {
					activeTickIndex = tickIndex
				} else {
					activeTickIndex = tickIndex - 1
				}

				initializedTicksCrossed += 1

				liquidityDelta := nextInitTick.LiquidityDelta
				liquidityDeltaAbs := new(big.Int).Abs(liquidityDelta)
				if (liquidityDelta.Sign() == 1) == isIncreasing {
					liquidity.Add(liquidity, liquidityDeltaAbs)
				} else {
					liquidity.Sub(liquidity, liquidityDeltaAbs)
				}
			}
		}
	}

	tickSpacingsCrossed := ekubomath.ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, sqrtRatio, p.key.Config.TickSpacing)

	var skipAhead uint32
	if initializedTicksCrossed != 0 {
		skipAhead = tickSpacingsCrossed / initializedTicksCrossed
	}

	return &quoting.Quote{
		ConsumedAmount:   amountRemaining.Sub(amount, amountRemaining),
		CalculatedAmount: calculatedAmount,
		FeesPaid:         feesPaid,
		Gas:              quoting.BaseGasCostOfOneConcentratedLiquidtitySwap + int64(initializedTicksCrossed)*quoting.GasCostOfOneInitializedTickCrossed + int64(tickSpacingsCrossed)*quoting.GasCostOfOneTickSpacingCrossed,
		SwapInfo: quoting.SwapInfo{
			SkipAhead: skipAhead,
			IsToken1:  isToken1,
			SwapStateAfter: &BasePoolSwapState{
				sqrtRatio,
				liquidity,
				activeTickIndex,
			},
			TickSpacingsCrossed: tickSpacingsCrossed,
		},
	}, nil
}
