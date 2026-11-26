package pools

import (
	"slices"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestConcentratedPoolQuote(t *testing.T) {
	t.Parallel()
	poolKey := func(tickSpacing uint32, fee uint64) *ConcentratedPoolKey {
		return NewPoolKey(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			common.HexToAddress("0x0000000000000000000000000000000000000001"),
			NewPoolConfig(common.Address{}, fee, NewConcentratedPoolTypeConfig(tickSpacing)),
		)
	}

	ticks := func(liquidity *uint256.Int) []Tick {
		return []Tick{
			{Number: math.MinTick, LiquidityDelta: big256.SInt256(liquidity)},
			{Number: math.MaxTick, LiquidityDelta: big256.SNeg(liquidity)},
		}
	}

	maxTickBounds := [2]int32{math.MinTick, math.MaxTick}

	t.Run("zero_liquidity_token1_input", func(t *testing.T) {
		p := NewConcentratedPool(
			poolKey(1, 0),
			NewConcentratedPoolState(
				NewConcentratedPoolSwapState(
					big256.U2Pow128,
					new(uint256.Int),
					0,
				),
				ticks(new(uint256.Int)),
				maxTickBounds,
				0,
			),
		)
		quote, err := p.Quote(big256.U1, true)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Sign())
	})

	t.Run("zero_liquidity_token0_input", func(t *testing.T) {
		p := NewConcentratedPool(
			poolKey(1, 0),
			NewConcentratedPoolState(
				NewConcentratedPoolSwapState(
					big256.U2Pow128,
					new(uint256.Int),
					0,
				),
				ticks(new(uint256.Int)),
				maxTickBounds,
				0,
			),
		)
		quote, err := p.Quote(big256.U1, false)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Sign())
	})

	t.Run("liquidity_token1_input", func(t *testing.T) {
		p := NewConcentratedPool(
			poolKey(1, 0),
			NewConcentratedPoolState(
				NewConcentratedPoolSwapState(
					big256.U2Pow128,
					big256.New("1000000000"),
					0,
				),
				[]Tick{
					{Number: 0, LiquidityDelta: int256.NewInt(1e9)},
					{Number: 1, LiquidityDelta: int256.NewInt(-1e9)},
				},
				maxTickBounds,
				0,
			),
		)
		quote, err := p.Quote(uint256.NewInt(1000), true)
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(499), quote.CalculatedAmount)
	})

	t.Run("liquidity_token0_input", func(t *testing.T) {
		p := NewConcentratedPool(
			poolKey(1, 0),
			NewConcentratedPoolState(
				NewConcentratedPoolSwapState(
					math.ToSqrtRatio(1),
					big256.New("1000000000"),
					0,
				),
				[]Tick{
					{Number: 0, LiquidityDelta: int256.NewInt(1e9)},
					{Number: 1, LiquidityDelta: int256.NewInt(-1e9)},
				},
				maxTickBounds,
				1,
			),
		)

		quote, err := p.Quote(uint256.NewInt(1000), false)
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(499), quote.CalculatedAmount)
	})
}

func TestNearestInitializedTickIndex(t *testing.T) {
	t.Parallel()
	t.Run("no ticks", func(t *testing.T) {
		require.Equal(t, -1, NearestInitializedTickIndex([]Tick{}, 0))
	})

	t.Run("index zero tick less than", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         -1,
				LiquidityDelta: (*int256.Int)(big256.U1),
			},
		}, 0))
	})

	t.Run("index zero tick equal to", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         0,
				LiquidityDelta: (*int256.Int)(big256.U1),
			},
		}, 0))
	})

	t.Run("index zero tick greater than", func(t *testing.T) {
		require.Equal(t, -1, NearestInitializedTickIndex([]Tick{
			{
				Number:         1,
				LiquidityDelta: (*int256.Int)(big256.U1),
			},
		}, 0))
	})

	t.Run("many ticks", func(t *testing.T) {
		ticks := []Tick{
			{
				Number:         -100,
				LiquidityDelta: new(int256.Int),
			},
			{
				Number:         -5,
				LiquidityDelta: new(int256.Int),
			},
			{
				Number:         -4,
				LiquidityDelta: new(int256.Int),
			},
			{
				Number:         18,
				LiquidityDelta: new(int256.Int),
			},
			{
				Number:         23,
				LiquidityDelta: new(int256.Int),
			},
			{
				Number:         50,
				LiquidityDelta: new(int256.Int),
			},
		}

		require.Equal(t, -1, NearestInitializedTickIndex(ticks, -101))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -100))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -99))
		require.Equal(t, 0, NearestInitializedTickIndex(ticks, -6))
		require.Equal(t, 1, NearestInitializedTickIndex(ticks, -5))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, -4))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, -3))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, 0))
		require.Equal(t, 2, NearestInitializedTickIndex(ticks, 17))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 18))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 19))
		require.Equal(t, 3, NearestInitializedTickIndex(ticks, 22))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 23))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 24))
		require.Equal(t, 4, NearestInitializedTickIndex(ticks, 49))
		require.Equal(t, 5, NearestInitializedTickIndex(ticks, 50))
		require.Equal(t, 5, NearestInitializedTickIndex(ticks, 51))
	})
}

func TestApproximateExtraDistinctTickBitmapLookupsWordBoundaries(t *testing.T) {
	t.Parallel()

	spacing := uint32(1)
	base := math.ToSqrtRatio(0)
	sameWord := math.ToSqrtRatio(128)
	nextWord := math.ToSqrtRatio(129)

	require.Equal(t, uint32(0), approximateExtraDistinctTickBitmapLookups(base, sameWord, spacing))
	require.Equal(t, uint32(1), approximateExtraDistinctTickBitmapLookups(base, nextWord, spacing))
}

func TestApproximateExtraDistinctTickBitmapLookupsNegativeTicks(t *testing.T) {
	t.Parallel()

	spacing := uint32(1)
	base := math.ToSqrtRatio(0)
	negativeSameWord := math.ToSqrtRatio(-1)
	negativePrevWord := math.ToSqrtRatio(-128)

	require.Equal(t, uint32(0), approximateExtraDistinctTickBitmapLookups(base, negativeSameWord, spacing))
	require.Equal(t, uint32(1), approximateExtraDistinctTickBitmapLookups(base, negativePrevWord, spacing))
}

func TestSuggestedSkipAhead(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint32(0), suggestedSkipAhead(0, 0))
	require.Equal(t, uint32(7), suggestedSkipAhead(0, 7))
	require.Equal(t, uint32(3), suggestedSkipAhead(3, 10))
	require.Equal(t, maxSkipAhead, suggestedSkipAhead(1, maxSkipAhead*2))
}

func TestAddLiquidityCutoffs(t *testing.T) {
	t.Parallel()
	var (
		checkedTickNumberBounds     = [2]int32{-2, 2}
		minCheckedTickNumber        = checkedTickNumberBounds[0]
		maxCheckedTickNumber        = checkedTickNumberBounds[1]
		minCheckedTickUninitialized = Tick{
			Number:         minCheckedTickNumber,
			LiquidityDelta: new(int256.Int),
		}
		maxCheckedTickUninitialized = Tick{
			Number:         maxCheckedTickNumber,
			LiquidityDelta: new(int256.Int),
		}
		betweenMinAndActiveTickNumber int32 = -1
		betweenActiveAndMaxTickNumber int32 = 1
		activeTickNumber              int32 = 0
		activeSqrtRatio                     = math.ToSqrtRatio(activeTickNumber)
		positiveLiquidity                   = uint256.NewInt(10)
	)

	newConcentratedPoolStateWithLiquidityCutoffs := func(liquidity *uint256.Int, ticks []Tick) *ConcentratedPoolState {
		state := NewConcentratedPoolState(
			NewConcentratedPoolSwapState(
				new(uint256.Int).Set(activeSqrtRatio),
				new(uint256.Int).Set(liquidity),
				-1,
			),
			ticks,
			checkedTickNumberBounds,
			activeTickNumber,
		)

		state.AddLiquidityCutoffs()

		return state
	}

	requireTicksEqual := func(t *testing.T, expected []Tick, actual []Tick) {
		require.True(t, slices.EqualFunc(expected, actual, func(e1, e2 Tick) bool {
			return e1.Number == e2.Number && e1.LiquidityDelta.Eq(e2.LiquidityDelta)
		}))
	}

	t.Run("empty_ticks", func(t *testing.T) {
		state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{})

		require.Equal(t, []Tick{minCheckedTickUninitialized, maxCheckedTickUninitialized}, state.SortedTicks)
		require.Equal(t, 0, state.ActiveTickIndex)
	})

	t.Run("positive_liqudity_delta", func(t *testing.T) {
		liquidityDelta := positiveLiquidity

		t.Run("initialized active tick", func(t *testing.T) {
			activeTickInitialized := Tick{
				Number:         activeTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{activeTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				activeTickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick", func(t *testing.T) {
			minCheckedTickInitialized := Tick{
				Number:         minCheckedTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{minCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized maxCheckedTick", func(t *testing.T) {
			maxCheckedTickInitialized := Tick{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{maxCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenMinAndActiveTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				tickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenActiveAndMaxTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				tickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})
	})

	t.Run("negative_liquidity_delta", func(t *testing.T) {
		liquidityDelta := new(uint256.Int).Neg(positiveLiquidity)

		t.Run("initialized active tick", func(t *testing.T) {
			activeTickInitialized := Tick{
				Number:         activeTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{activeTickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
				activeTickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick", func(t *testing.T) {
			minCheckedTickInitialized := Tick{
				Number:         minCheckedTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{minCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized maxCheckedTick", func(t *testing.T) {
			maxCheckedTickInitialized := Tick{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{maxCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
				maxCheckedTickInitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenMinAndActiveTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(new(uint256.Int), []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
				tickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenActiveAndMaxTickNumber,
				LiquidityDelta: big256.SInt256(liquidityDelta),
			}

			state := newConcentratedPoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: big256.SNeg(liquidityDelta),
				},
				tickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})
	})
}

// The fetched state describes liquidity only for ticks in [TickBounds[0], TickBounds[1]): AddLiquidityCutoffs adds all
// liquidity below the window at TickBounds[0] and removes it at TickBounds[1].
func TestActiveTickOutsideTickBounds(t *testing.T) {
	t.Parallel()

	bounds := [2]int32{-227050, -216810}
	for _, tc := range []struct {
		activeTick int32
		outside    bool
	}{
		{-227148, true},
		{-227051, true},
		{-227050, false},
		{-227046, false},
		{-216811, false},
		{-216810, true},
		{-216000, true},
	} {
		state := NewConcentratedPoolState(nil, nil, bounds, tc.activeTick)
		require.Equal(t, tc.outside, state.ActiveTickOutsideTickBounds(), "active tick %d", tc.activeTick)
	}

	// A window clamped to the tick range holds a price at the range's end: no fetch can return a wider one.
	for _, activeTick := range []int32{math.MinTick, math.MaxTick} {
		state := NewConcentratedPoolState(nil, nil, [2]int32{math.MinTick, math.MaxTick}, activeTick)
		require.False(t, state.ActiveTickOutsideTickBounds(), "active tick %d", activeTick)
	}
}

// windowPool is a concentrated pool whose state is known on ticks [-1000, 1000] (the tracker's tick bounds), with
// liquidity L there and none outside it once the cutoffs are added, at tick 0, tick spacing 1, fee 0.
func windowPool(t *testing.T) *ConcentratedPool {
	t.Helper()
	s := NewConcentratedPoolState(NewConcentratedPoolSwapState(big256.U2Pow128.Clone(), uint256.NewInt(1e18), -1),
		nil, [2]int32{-1000, 1000}, 0)
	s.AddLiquidityCutoffs()
	return NewConcentratedPool(NewPoolKey(common.HexToAddress("0x0"), common.HexToAddress("0x1"),
		NewPoolConfig(common.Address{}, 0, NewConcentratedPoolTypeConfig(1))), s)
}

// Past the pool's capacity in the known window, a quote stops where the executed swap stops, at SwapInfo's
// price limit (one tick past the bound): it returns what it consumed, with the rest left over, a gas estimate for
// that swap and the state at the limit. Before, it walked the price on to the global limit through empty ticks,
// counting a bitmap lookup per word on the way (hundreds of millions of gas), and left the state there.
func TestConcentratedQuotePastTheWindowStopsAtThePriceLimit(t *testing.T) {
	for _, isToken1 := range []bool{false, true} {
		p := windowPool(t)
		over, err := p.Quote(uint256.NewInt(1e18), isToken1) // far past the window's capacity
		require.NoError(t, err)
		require.True(t, over.ConsumedAmount.Lt(uint256.NewInt(1e18)), "token1 %v: a partial fill must leave input", isToken1)
		full, err := windowPool(t).Quote(over.ConsumedAmount, isToken1) // exactly what the window holds
		require.NoError(t, err)
		require.True(t, full.CalculatedAmount.Eq(over.CalculatedAmount), "token1 %v: the same output", isToken1)

		t.Logf("token1 %v: consumed %v of 1e18, gas %d past the window, %d at capacity", isToken1,
			over.ConsumedAmount, over.Gas, full.Gas)
		bound := map[bool]int32{false: -1000 - 1, true: 1000 + 1}[isToken1]
		limit := math.ToSqrtRatio(bound)
		after := over.SwapInfo.SwapStateAfter.(*ConcentratedPoolSwapState)
		require.True(t, after.SqrtRatio.Eq(limit), "token1 %v: the state after is at %v, want the limit %v",
			isToken1, after.SqrtRatio, limit)
		require.LessOrEqual(t, over.Gas, full.Gas+quoting.GasCostOfOneExtraTickBitmapSload+
			quoting.GasCostOfOneExtraConcentratedMathRound, "token1 %v: gas %d against %d at capacity", isToken1,
			over.Gas, full.Gas)
	}
}
