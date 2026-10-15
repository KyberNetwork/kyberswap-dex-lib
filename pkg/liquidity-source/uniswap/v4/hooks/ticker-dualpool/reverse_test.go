package tickerdualpool

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestGrossBeforeOutputFeesMatchesExhaustiveFloors(t *testing.T) {
	for _, rates := range [][2]uint64{{0, 0}, {100, 0}, {0, 2000}, {100, 50}, {400, 400}, {1000, 999}, {1000, 1000}} {
		t.Run(fmt.Sprintf("%d_%d", rates[0], rates[1]), func(t *testing.T) {
			for desired := uint64(1); desired <= 512; desired++ {
				var expected uint64
				for gross := uint64(1); gross <= 640; gross++ {
					if gross-gross*rates[0]/10000-gross*rates[1]/10000 >= desired {
						expected = gross
						break
					}
				}
				actual, err := grossBeforeOutputFees(uint256.NewInt(desired), rates[0], rates[1])
				require.NoError(t, err)
				require.Equal(t, expected, actual.Uint64(), "desired %d", desired)
			}
		})
	}
	// The greatest on-chain gross output remains reachable even if the safe
	// combined-fee bound would exceed int128 by a rounding unit.
	maxGross := new(uint256.Int).Rsh(uMaxUint128, 1)
	_, _, desired := outputFees(maxGross, 1000, 1000)
	gross, err := grossBeforeOutputFees(&desired, 1000, 1000)
	require.NoError(t, err)
	require.LessOrEqual(t, gross.Cmp(maxGross), 0)
}

func TestReverseQuoteHandlesPriceStepsAndSeparateFeeDips(t *testing.T) {
	for _, direction := range []bool{true, false} {
		s := fixtureState()
		s.HookFeeBps, s.CreatorTaxBps = 400, 400
		// Unequal token prices can skip a qualifying gross output and land on
		// a fee dip: gross 24 leaves 24, but gross 25 leaves only 23.
		s.Pool.SqrtPriceX96 = uint256.MustFromDecimal("112045541949572279837463876454")
		s.Pool.Tick = 6931
		var forward []Quote
		for input := uint64(1); input <= 256; input++ {
			q, err := QuoteExactInput(s, direction, uint256.NewInt(input))
			if err == nil {
				forward = append(forward, q)
			}
		}
		for _, desired := range []uint64{1, 23, 24, 25, 46, 47, 48, 91} {
			var expected *uint256.Int
			for _, q := range forward {
				if q.NetOutput.CmpUint64(desired) >= 0 {
					expected = q.Swap.AmountIn
					break
				}
			}
			require.NotNil(t, expected)
			q, err := QuoteRequiredInput(s, direction, uint256.NewInt(desired))
			require.NoError(t, err)
			require.Equal(t, expected.Dec(), q.Swap.AmountIn.Dec(), "direction %t desired %d", direction, desired)
			require.GreaterOrEqual(t, q.NetOutput.CmpUint64(desired), 0)
		}
	}
}

func TestReverseHookFeeDeltaAndParentStateUpdates(t *testing.T) {
	for _, direction := range []bool{true, false} {
		h := poolSnapshotFixture()
		params := &uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: direction, AmountSpecified: big.NewInt(198)}
		r, err := h.BeforeSwap(params)
		require.NoError(t, err)
		q := r.SwapInfo.(Quote)
		require.Equal(t, new(big.Int).Sub(q.Swap.AmountOut.ToBig(), params.AmountSpecified).String(), r.DeltaSpecified.String())
		require.Equal(t, q.Swap.AmountIn.ToBig().String(), r.DeltaUnspecified.String())

		sim := localSimulator(t, h.Snapshot)
		inputSide, outputSide := 0, 1
		if !direction {
			inputSide, outputSide = 1, 0
		}
		reverse, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: sim.GetTokens()[outputSide], Amount: params.AmountSpecified}, TokenIn: sim.GetTokens()[inputSide]})
		require.NoError(t, err)
		forward, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *reverse.TokenAmountIn, TokenOut: sim.GetTokens()[outputSide]})
		require.NoError(t, err)
		require.GreaterOrEqual(t, forward.TokenAmountOut.Amount.Cmp(params.AmountSpecified), 0)
		info := reverse.SwapInfo.(uniswapv4.SwapInfo).HookSwapInfo.(Quote)
		clone := sim.CloneState().(*uniswapv4.PoolSimulator)
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *reverse.TokenAmountIn, TokenAmountOut: *forward.TokenAmountOut, SwapInfo: reverse.SwapInfo})
		clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *reverse.TokenAmountIn, TokenAmountOut: *forward.TokenAmountOut, SwapInfo: forward.SwapInfo})
		require.Equal(t, info.Swap.Balance0.Dec(), sim.GetReserves()[0].String())
		require.Equal(t, info.Swap.Balance1.Dec(), sim.GetReserves()[1].String())
		require.Equal(t, clone.V3Pool.TickCurrent, sim.V3Pool.TickCurrent)
		require.Zero(t, clone.V3Pool.SqrtRatioX96.Cmp(&sim.V3Pool.SqrtRatioX96))
		_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *reverse.TokenAmountIn, TokenOut: sim.GetTokens()[outputSide]})
		require.NoError(t, err)
	}
}

func TestReverseRejectsImpossibleFeesAndUnreachableOutput(t *testing.T) {
	for _, rates := range [][2]uint16{{1001, 0}, {1000, 1001}, {0, 2001}} {
		s := fixtureState()
		s.HookFeeBps, s.CreatorTaxBps = rates[0], rates[1]
		_, err := QuoteRequiredInput(s, true, uint256.NewInt(100))
		require.ErrorIs(t, err, ErrInvalidState)
	}
	s := fixtureState()
	_, err := QuoteRequiredInput(s, true, s.Total1)
	require.ErrorIs(t, err, uniswapv4.ErrAmountOutUnreachable)
	_, err = QuoteRequiredInput(s, true, new(uint256.Int))
	require.ErrorIs(t, err, ErrInvalidState)
	// Keep a valid state for the no-fee registration branch.
	s.Registered = false
	s.HookFeeBps, s.CreatorTaxBps = 65535, 65535
	q, err := QuoteRequiredInput(s, true, uint256.NewInt(100))
	require.NoError(t, err)
	require.True(t, q.HookFee.IsZero())
	require.True(t, q.CreatorTax.IsZero())
}
