package quoting_test

import (
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

var (
	checkedTickNumberBounds     = [2]int32{-2, 2}
	minCheckedTickNumber        = checkedTickNumberBounds[0]
	maxCheckedTickNumber        = checkedTickNumberBounds[1]
	minCheckedTickUninitialized = quoting2.Tick{
		Number:         minCheckedTickNumber,
		LiquidityDelta: new(big.Int),
	}
	maxCheckedTickUninitialized = quoting2.Tick{
		Number:         maxCheckedTickNumber,
		LiquidityDelta: new(big.Int),
	}
	betweenMinAndActiveTickNumber int32 = -1
	betweenActiveAndMaxTickNumber int32 = 1
	activeTickNumber              int32 = 0
	positiveLiquidity                   = big.NewInt(10)
)

func newPoolState(liquidity *big.Int, ticks []quoting2.Tick) quoting2.PoolState {
	return quoting2.NewPoolState(
		liquidity,
		math.ToSqrtRatio(activeTickNumber),
		activeTickNumber,
		ticks,
		checkedTickNumberBounds,
	)
}

func requireTicksEqual(t *testing.T, expected []quoting2.Tick, actual []quoting2.Tick) {
	require.True(t, slices.EqualFunc(expected, actual, func(e1, e2 quoting2.Tick) bool {
		return e1.Number == e2.Number && e1.LiquidityDelta.Cmp(e2.LiquidityDelta) == 0
	}))
}

func TestEmptyTicks(t *testing.T) {
	state := newPoolState(new(big.Int), []quoting2.Tick{})

	require.Equal(t, []quoting2.Tick{minCheckedTickUninitialized, maxCheckedTickUninitialized}, state.Ticks)
}

func TestPositiveLiquidityDelta(t *testing.T) {
	liquidityDelta := positiveLiquidity

	t.Run("initialized active tick", func(t *testing.T) {
		activeTickInitialized := quoting2.Tick{
			Number:         activeTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(positiveLiquidity, []quoting2.Tick{activeTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickUninitialized,
			activeTickInitialized,
			{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
		}, state.Ticks)
	})

	t.Run("initialized minCheckedTick", func(t *testing.T) {
		minCheckedTickInitialized := quoting2.Tick{
			Number:         minCheckedTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(positiveLiquidity, []quoting2.Tick{minCheckedTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickInitialized,
			{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
		}, state.Ticks)
	})

	t.Run("initialized maxCheckedTick", func(t *testing.T) {
		maxCheckedTickInitialized := quoting2.Tick{
			Number:         maxCheckedTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(new(big.Int), []quoting2.Tick{maxCheckedTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickUninitialized,
			maxCheckedTickUninitialized,
		}, state.Ticks)
	})

	t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
		tickInitialized := quoting2.Tick{
			Number:         betweenMinAndActiveTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(positiveLiquidity, []quoting2.Tick{tickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickUninitialized,
			tickInitialized,
			{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
		}, state.Ticks)
	})

	t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
		tickInitialized := quoting2.Tick{
			Number:         betweenActiveAndMaxTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(new(big.Int), []quoting2.Tick{tickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickUninitialized,
			tickInitialized,
			{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
		}, state.Ticks)
	})
}

func TestNegativeLiquidityDelta(t *testing.T) {
	liquidityDelta := new(big.Int).Neg(positiveLiquidity)

	t.Run("initialized active tick", func(t *testing.T) {
		activeTickInitialized := quoting2.Tick{
			Number:         activeTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(new(big.Int), []quoting2.Tick{activeTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
			activeTickInitialized,
			maxCheckedTickUninitialized,
		}, state.Ticks)
	})

	t.Run("initialized minCheckedTick", func(t *testing.T) {
		minCheckedTickInitialized := quoting2.Tick{
			Number:         minCheckedTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(new(big.Int), []quoting2.Tick{minCheckedTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			minCheckedTickUninitialized,
			maxCheckedTickUninitialized,
		}, state.Ticks)
	})

	t.Run("initialized maxCheckedTick", func(t *testing.T) {
		maxCheckedTickInitialized := quoting2.Tick{
			Number:         maxCheckedTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(positiveLiquidity, []quoting2.Tick{maxCheckedTickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
			maxCheckedTickInitialized,
		}, state.Ticks)
	})

	t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
		tickInitialized := quoting2.Tick{
			Number:         betweenMinAndActiveTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(new(big.Int), []quoting2.Tick{tickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
			tickInitialized,
			maxCheckedTickUninitialized,
		}, state.Ticks)
	})

	t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
		tickInitialized := quoting2.Tick{
			Number:         betweenActiveAndMaxTickNumber,
			LiquidityDelta: new(big.Int).Set(liquidityDelta),
		}

		state := newPoolState(positiveLiquidity, []quoting2.Tick{tickInitialized})

		requireTicksEqual(t, []quoting2.Tick{
			{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Neg(liquidityDelta),
			},
			tickInitialized,
			maxCheckedTickUninitialized,
		}, state.Ticks)
	})
}
