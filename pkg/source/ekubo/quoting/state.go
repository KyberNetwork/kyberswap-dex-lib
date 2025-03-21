package quoting

import (
	"math/big"
	"slices"
)

type PoolState struct {
	Liquidity  *big.Int `json:"liquidity"`
	SqrtRatio  *big.Int `json:"sqrtRatio"`
	ActiveTick int32    `json:"activeTick"`
	Ticks      []Tick   `json:"ticks"`
	TickBounds [2]int32 `json:"tickBounds"`
}

func NewPoolState(liquidity, sqrtRatio *big.Int, activeTick int32, ticks []Tick, tickBounds [2]int32) PoolState {
	state := PoolState{
		Liquidity:  liquidity,
		SqrtRatio:  sqrtRatio,
		ActiveTick: activeTick,
		Ticks:      ticks,
		TickBounds: tickBounds,
	}
	state.addLiquidityCutoffs()

	return state
}

func (s *PoolState) UpdateTick(updatedTickNumber int32, liquidityDelta *big.Int, upper, forceInsert bool) {
	ticks := s.Ticks

	liquidityDelta = new(big.Int).Set(liquidityDelta)
	if upper {
		liquidityDelta.Neg(liquidityDelta)
	}

	nearestTickIndex := NearestInitializedTickIndex(ticks, updatedTickNumber)

	var (
		nearestTick       *Tick
		nearestTickNumber = invalidTickNumber
	)

	if nearestTickIndex != InvalidTickIndex {
		nearestTick = &ticks[nearestTickIndex]
		nearestTickNumber = nearestTick.Number
	}

	newTickReferenced := nearestTickNumber != updatedTickNumber

	if newTickReferenced {
		if !forceInsert && nearestTickIndex == InvalidTickIndex {
			delta := ticks[0].LiquidityDelta
			delta.Add(delta, liquidityDelta)
		} else if !forceInsert && nearestTickIndex == len(ticks)-1 {
			delta := ticks[len(ticks)-1].LiquidityDelta
			delta.Add(delta, liquidityDelta)
		} else {
			var insertIdx int
			if nearestTickIndex != InvalidTickIndex {
				insertIdx = nearestTickIndex + 1
			}

			s.Ticks = slices.Insert(ticks, insertIdx, Tick{
				Number:         updatedTickNumber,
				LiquidityDelta: liquidityDelta,
			})
		}
	} else {
		newDelta := new(big.Int).Add(nearestTick.LiquidityDelta, liquidityDelta)

		if newDelta.Sign() == 0 && !slices.Contains(s.TickBounds[:], nearestTickNumber) {
			s.Ticks = slices.Delete(ticks, nearestTickIndex, nearestTickIndex+1)
		} else {
			nearestTick.LiquidityDelta = newDelta
		}
	}
}

func (s *PoolState) addLiquidityCutoffs() {
	currentLiquidity := new(big.Int)
	belowActiveTick := true

	// The liquidity added/removed by out-of-range initialized ticks (i.e. lower than the min checked tick number)
	liquidityDeltaMin := new(big.Int)

	for _, tick := range s.Ticks {
		if belowActiveTick && s.ActiveTick < tick.Number {
			belowActiveTick = false

			liquidityDeltaMin.Sub(s.Liquidity, currentLiquidity)

			// We now need to switch to tracking the liquidity that needs to be cut off at the max checked tick number, therefore reset to the actual liquidity
			currentLiquidity.Set(s.Liquidity)
		}

		currentLiquidity.Add(currentLiquidity, tick.LiquidityDelta)
	}

	if belowActiveTick {
		liquidityDeltaMin.Sub(s.Liquidity, currentLiquidity)
		currentLiquidity.Set(s.Liquidity)
	}

	s.UpdateTick(s.TickBounds[0], liquidityDeltaMin, false, true)
	s.UpdateTick(s.TickBounds[1], currentLiquidity, true, true)
}
