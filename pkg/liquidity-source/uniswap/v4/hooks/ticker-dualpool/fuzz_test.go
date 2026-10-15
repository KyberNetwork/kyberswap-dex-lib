package tickerdualpool

import (
	"encoding/json"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func rangeBuckets(config int) []dualpool.Bucket {
	switch config {
	case 0:
		result := make([]dualpool.Bucket, 8)
		for i := range result {
			width := int32(1000 + 1000*i)
			result[i] = dualpool.Bucket{TickLower: -width, TickUpper: width, WeightBps: 1250}
		}
		return result
	case 1:
		return []dualpool.Bucket{{TickLower: -1000, TickUpper: 1000, WeightBps: 4000}, {TickLower: -10000, TickUpper: -2000, WeightBps: 3000}, {TickLower: 2000, TickUpper: 10000, WeightBps: 3000}}
	default:
		return []dualpool.Bucket{{TickLower: -1000, TickUpper: 1000, WeightBps: 5000}, {TickLower: -1000, TickUpper: 1000, WeightBps: 5000}}
	}
}

func FuzzQuoteConservationAndPurity(f *testing.F) {
	for _, amount := range []uint64{1, 1001, 100000000000000000, 900000000000000000, 1999999999999999999} {
		f.Add(amount, true, uint8(0))
		f.Add(amount, false, uint8(1))
	}
	f.Fuzz(func(t *testing.T, amount uint64, direction bool, config uint8) {
		state := fixtureState()
		state.Pool.Buckets = rangeBuckets(int(config % 3))
		input := new(uint256.Int).Mul(uint256.NewInt(amount%2000000000000000000+1), uint256.NewInt(1000000))
		before, err := json.Marshal(state)
		require.NoError(t, err)
		q, quoteErr := QuoteExactInput(state, direction, input)
		after, err := json.Marshal(state)
		require.NoError(t, err)
		require.Equal(t, string(before), string(after))
		if quoteErr != nil {
			return
		}
		require.False(t, q.NetOutput.IsZero())
		require.LessOrEqual(t, q.Swap.AmountIn.Cmp(input), 0)
		gross := new(uint256.Int).Add(q.NetOutput, q.HookFee)
		gross.Add(gross, q.CreatorTax)
		require.Equal(t, q.Swap.AmountOut.Dec(), gross.Dec())
		originalIn, originalOut := state.Total0, state.Total1
		afterIn, afterOut := q.Swap.Balance0, q.Swap.Balance1
		if !direction {
			originalIn, originalOut = originalOut, originalIn
			afterIn, afterOut = afterOut, afterIn
		}
		require.LessOrEqual(t, q.Swap.AmountOut.Cmp(originalOut), 0)
		require.LessOrEqual(t, afterIn.Cmp(new(uint256.Int).Add(originalIn, q.Swap.AmountIn)), 0, "pool cannot create input tokens")
		require.LessOrEqual(t, afterOut.Cmp(new(uint256.Int).Sub(originalOut, q.Swap.AmountOut)), 0, "pool cannot create output tokens")
	})
}

func FuzzReverseQuoteReachesKnownOutput(f *testing.F) {
	f.Add(uint32(1000), true, uint8(0))
	f.Add(uint32(800000), false, uint8(1))
	f.Fuzz(func(t *testing.T, size uint32, direction bool, config uint8) {
		s := fixtureState()
		s.Pool.Buckets = rangeBuckets(int(config % 3))
		input := new(uint256.Int).Mul(uint256.NewInt(uint64(size%900000+1)), uint256.MustFromDecimal("1000000000000000000"))
		known, err := QuoteExactInput(s, direction, input)
		require.NoError(t, err)
		reversed, err := QuoteRequiredInput(s, direction, known.NetOutput)
		require.NoError(t, err)
		require.LessOrEqual(t, reversed.Swap.AmountIn.Cmp(input), 0, "a known sufficient input bounds reverse search")
		actual, err := QuoteExactInput(s, direction, reversed.Swap.AmountIn)
		require.NoError(t, err)
		require.GreaterOrEqual(t, actual.NetOutput.Cmp(known.NetOutput), 0)
	})
}
