package dexv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestBorrowExchangePriceRoundsGrowthUp(t *testing.T) {
	// 1e12 * 1 * 1 / 315360000000 has a nonzero remainder: ceil = 4.
	var packed, field big.Int
	packed.SetInt64(1)
	for _, f := range []struct {
		value int64
		shift uint
	}{{1_000_000_000_000, BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE}, {1_000_000_000_000, BITS_EXCHANGE_PRICES_BORROW_EXCHANGE_PRICE}, {100, BITS_EXCHANGE_PRICES_LAST_TIMESTAMP}, {1, BITS_EXCHANGE_PRICES_SUPPLY_RATIO}} {
		field.SetInt64(f.value)
		field.Lsh(&field, f.shift)
		packed.Or(&packed, &field)
	}
	supply, borrow, err := _calcExchangePrice(&packed, 101)
	require.NoError(t, err)
	require.Equal(t, "1000000000000", supply.String())
	require.Equal(t, "1000000000004", borrow.String())
	// Exact division and no elapsed time must not receive an extra unit.
	_, borrow, err = _calcExchangePrice(&packed, 100+31536)
	require.NoError(t, err)
	require.Equal(t, "1000000100000", borrow.String())
	_, borrow, err = _calcExchangePrice(&packed, 100)
	require.NoError(t, err)
	require.Equal(t, "1000000000000", borrow.String())
}

func TestSwapPriceReferenceAcrossEmptyRegions(t *testing.T) {
	for _, zeroForOne := range []bool{true, false} {
		for _, startsEmpty := range []bool{true, false} {
			t.Run(fmt.Sprintf("zeroForOne=%t/startsEmpty=%t", zeroForOne, startsEmpty), func(t *testing.T) {
				sim, _ := snapshotSimulator(t)
				p := *sim.V3Pool
				p.dynamicVersion = false
				p.tickSpacing = 10
				p.TickCurrent = 0
				p.SqrtRatioX96 = new(uint256.Int).Lsh(uint256.NewInt(1), 96)
				p.Liquidity = uint256.NewInt(0)
				const l = 1_000_000_000_000
				ticks := []v3Entities.Tick{
					{Index: -10010, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(l)},
					{Index: -10000, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(-l)},
					{Index: 10000, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(l)},
					{Index: 10010, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(-l)},
				}
				if !startsEmpty {
					p.Liquidity.SetUint64(l)
					ticks = append(ticks, v3Entities.Tick{Index: -10, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(l)}, v3Entities.Tick{Index: 10, LiquidityGross: uint256.NewInt(l), LiquidityNet: int256.NewInt(-l)})
					// Provider validates sorted ticks.
					ticks[2], ticks[3], ticks[4], ticks[5] = ticks[4], ticks[5], ticks[2], ticks[3]
				}
				provider, err := v3Entities.NewTickListDataProvider(ticks, 10)
				require.NoError(t, err)
				p.TickDataProvider = provider
				amount := int64(1000)
				if !startsEmpty {
					amount = 600_000_000
				}
				q, err := p.GetOutputAmountV2(int256.NewInt(amount), zeroForOne, nil)
				require.NoError(t, err)
				require.Positive(t, q.ReturnedAmount.Sign())
				if startsEmpty {
					firstTick := 10000
					if zeroForOne {
						firstTick = -10000
					}
					var want uint256.Int
					require.NoError(t, utils.GetSqrtRatioAtTickV2(firstTick, &want))
					require.True(t, want.Eq(&q.sqrtPriceReference))
					require.NoError(t, _verifySqrtPriceX96ChangeLimits(&q.sqrtPriceReference, q.SqrtRatioX96))
					require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(p.SqrtRatioX96, q.SqrtRatioX96), ErrSqrtPriceChangeOutOfBounds)
				} else {
					require.GreaterOrEqual(t, q.CrossInitTickLoops, 2)
					require.True(t, p.SqrtRatioX96.Eq(&q.sqrtPriceReference))
					require.ErrorIs(t, _verifySqrtPriceX96ChangeLimits(&q.sqrtPriceReference, q.SqrtRatioX96), ErrSqrtPriceChangeOutOfBounds)
				}
			})
		}
	}
}

func TestSwapRejectsNonPositiveInput(t *testing.T) {
	p, _ := snapshotSimulator(t)
	for _, amount := range []int64{-1, 0} {
		_, err := p.V3Pool.GetOutputAmountV2(int256.NewInt(amount), true, nil)
		require.ErrorIs(t, err, ErrAmountOutOfLimits)
	}
}

func TestUnknownLiquidityMergeKeyIsConservative(t *testing.T) {
	p := testLiquiditySimulator()
	for _, key := range []string{"", "unknown", "fluid:token:futureField"} {
		require.Equal(t, pool.SwapLimitMergeMin, p.GetLimitMergePolicy(key))
	}
	for field, name := range liquidityFieldNames {
		require.Equal(t, liquidityMergePolicy(field), p.GetLimitMergePolicy("fluid:token:"+name))
	}
}
