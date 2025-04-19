package pools

import (
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestBasePoolQuote(t *testing.T) {
	poolKey := func(tickSpacing uint32, fee uint64) *PoolKey {
		return NewPoolKey(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			common.HexToAddress("0x0000000000000000000000000000000000000001"),
			PoolConfig{
				Fee:         fee,
				TickSpacing: tickSpacing,
				Extension:   common.Address{},
			},
		)
	}

	ticks := func(liquidity *big.Int) []Tick {
		return []Tick{
			{Number: math.MinTick, LiquidityDelta: new(big.Int).Set(liquidity)},
			{Number: math.MaxTick, LiquidityDelta: new(big.Int).Neg(liquidity)},
		}
	}

	maxTickBounds := [2]int32{math.MinTick, math.MaxTick}

	t.Run("zero_liquidity_token1_input", func(t *testing.T) {
		p := NewBasePool(
			poolKey(1, 0),
			&BasePoolState{
				BasePoolSwapState: &BasePoolSwapState{
					SqrtRatio:       math.TwoPow128,
					Liquidity:       new(big.Int),
					ActiveTickIndex: 0,
				},
				SortedTicks: ticks(new(big.Int)),
				TickBounds:  maxTickBounds,
				ActiveTick:  0,
			},
		)
		quote, err := p.Quote(bignum.One, true)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Sign())
	})

	t.Run("zero_liquidity_token0_input", func(t *testing.T) {
		p := NewBasePool(
			poolKey(1, 0),
			&BasePoolState{
				BasePoolSwapState: &BasePoolSwapState{
					SqrtRatio:       math.TwoPow128,
					Liquidity:       new(big.Int),
					ActiveTickIndex: 0,
				},
				SortedTicks: ticks(new(big.Int)),
				TickBounds:  maxTickBounds,
				ActiveTick:  0,
			},
		)
		quote, err := p.Quote(bignum.One, true)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Sign())
	})

	t.Run("liquidity_token1_input", func(t *testing.T) {
		p := NewBasePool(
			poolKey(1, 0),
			&BasePoolState{
				BasePoolSwapState: &BasePoolSwapState{
					SqrtRatio:       math.TwoPow128,
					Liquidity:       bignum.NewBig("1_000_000_000"),
					ActiveTickIndex: 0,
				},
				SortedTicks: []Tick{
					{Number: 0, LiquidityDelta: bignum.NewBig("1_000_000_000")},
					{Number: 1, LiquidityDelta: bignum.NewBig("-1_000_000_000")},
				},
				TickBounds: maxTickBounds,
				ActiveTick: 0,
			},
		)
		quote, err := p.Quote(big.NewInt(1000), true)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(499)))
	})

	t.Run("liquidity_token0_input", func(t *testing.T) {
		p := NewBasePool(
			poolKey(1, 0),
			&BasePoolState{
				BasePoolSwapState: &BasePoolSwapState{
					SqrtRatio:       math.ToSqrtRatio(1),
					Liquidity:       bignum.NewBig("1_000_000_000"),
					ActiveTickIndex: 0,
				},
				SortedTicks: []Tick{
					{Number: 0, LiquidityDelta: bignum.NewBig("1_000_000_000")},
					{Number: 1, LiquidityDelta: bignum.NewBig("-1_000_000_000")},
				},
				TickBounds: maxTickBounds,
				ActiveTick: 1,
			},
		)

		quote, err := p.Quote(big.NewInt(1000), false)
		require.NoError(t, err)

		require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(499)))
	})
}

func TestNearestInitializedTickIndex(t *testing.T) {
	t.Run("no ticks", func(t *testing.T) {
		require.Equal(t, -1, NearestInitializedTickIndex([]Tick{}, 0))
	})

	t.Run("index zero tick less than", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         -1,
				LiquidityDelta: bignum.One,
			},
		}, 0))
	})

	t.Run("index zero tick equal to", func(t *testing.T) {
		require.Equal(t, 0, NearestInitializedTickIndex([]Tick{
			{
				Number:         0,
				LiquidityDelta: bignum.One,
			},
		}, 0))
	})

	t.Run("index zero tick greater than", func(t *testing.T) {
		require.Equal(t, -1, NearestInitializedTickIndex([]Tick{
			{
				Number:         1,
				LiquidityDelta: bignum.One,
			},
		}, 0))
	})

	t.Run("many ticks", func(t *testing.T) {
		ticks := []Tick{
			{
				Number:         -100,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         -5,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         -4,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         18,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         23,
				LiquidityDelta: new(big.Int),
			},
			{
				Number:         50,
				LiquidityDelta: new(big.Int),
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

func TestAddLiquidityCutoffs(t *testing.T) {
	var (
		checkedTickNumberBounds     = [2]int32{-2, 2}
		minCheckedTickNumber        = checkedTickNumberBounds[0]
		maxCheckedTickNumber        = checkedTickNumberBounds[1]
		minCheckedTickUninitialized = Tick{
			Number:         minCheckedTickNumber,
			LiquidityDelta: new(big.Int),
		}
		maxCheckedTickUninitialized = Tick{
			Number:         maxCheckedTickNumber,
			LiquidityDelta: new(big.Int),
		}
		betweenMinAndActiveTickNumber int32 = -1
		betweenActiveAndMaxTickNumber int32 = 1
		activeTickNumber              int32 = 0
		activeSqrtRatio                     = math.ToSqrtRatio(activeTickNumber)
		positiveLiquidity                   = big.NewInt(10)
	)

	newBasePoolStateWithLiquidityCutoffs := func(liquidity *big.Int, ticks []Tick) *BasePoolState {
		state := BasePoolState{
			BasePoolSwapState: &BasePoolSwapState{
				SqrtRatio:       new(big.Int).Set(activeSqrtRatio),
				Liquidity:       new(big.Int).Set(liquidity),
				ActiveTickIndex: -1, // Will be filled in
			},
			SortedTicks: ticks,
			TickBounds:  checkedTickNumberBounds,
			ActiveTick:  activeTickNumber,
		}

		state.AddLiquidityCutoffs()

		return &state
	}

	requireTicksEqual := func(t *testing.T, expected []Tick, actual []Tick) {
		require.True(t, slices.EqualFunc(expected, actual, func(e1, e2 Tick) bool {
			return e1.Number == e2.Number && e1.LiquidityDelta.Cmp(e2.LiquidityDelta) == 0
		}))
	}

	t.Run("empty_ticks", func(t *testing.T) {
		state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{})

		require.Equal(t, []Tick{minCheckedTickUninitialized, maxCheckedTickUninitialized}, state.SortedTicks)
		require.Equal(t, 0, state.ActiveTickIndex)
	})

	t.Run("positive_liqudity_delta", func(t *testing.T) {
		liquidityDelta := positiveLiquidity

		t.Run("initialized active tick", func(t *testing.T) {
			activeTickInitialized := Tick{
				Number:         activeTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{activeTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				activeTickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick", func(t *testing.T) {
			minCheckedTickInitialized := Tick{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{minCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized maxCheckedTick", func(t *testing.T) {
			maxCheckedTickInitialized := Tick{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{maxCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenMinAndActiveTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				tickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenActiveAndMaxTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				tickInitialized,
				{
					Number:         maxCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})
	})

	t.Run("negative_liquidity_delta", func(t *testing.T) {
		liquidityDelta := new(big.Int).Neg(positiveLiquidity)

		t.Run("initialized active tick", func(t *testing.T) {
			activeTickInitialized := Tick{
				Number:         activeTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{activeTickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
				activeTickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick", func(t *testing.T) {
			minCheckedTickInitialized := Tick{
				Number:         minCheckedTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{minCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				minCheckedTickUninitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized maxCheckedTick", func(t *testing.T) {
			maxCheckedTickInitialized := Tick{
				Number:         maxCheckedTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{maxCheckedTickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
				maxCheckedTickInitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})

		t.Run("initialized minCheckedTick < tick < activeTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenMinAndActiveTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(new(big.Int), []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
				tickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 1, state.ActiveTickIndex)
		})

		t.Run("initialized activeTick < tick < maxCheckedTick", func(t *testing.T) {
			tickInitialized := Tick{
				Number:         betweenActiveAndMaxTickNumber,
				LiquidityDelta: new(big.Int).Set(liquidityDelta),
			}

			state := newBasePoolStateWithLiquidityCutoffs(positiveLiquidity, []Tick{tickInitialized})

			requireTicksEqual(t, []Tick{
				{
					Number:         minCheckedTickNumber,
					LiquidityDelta: new(big.Int).Neg(liquidityDelta),
				},
				tickInitialized,
				maxCheckedTickUninitialized,
			}, state.SortedTicks)
			require.Equal(t, 0, state.ActiveTickIndex)
		})
	})
}
