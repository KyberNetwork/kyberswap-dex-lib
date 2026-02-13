package pools

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestBoostedFeesPoolQuoteSameTimestampNoop(t *testing.T) {
	t.Parallel()

	pool := testBoostedFeesPool(100, 0, 0, nil)
	quote, err := pool.quoteWithTimestampFn(uint256.NewInt(0), false, func() uint64 { return 100 })
	require.NoError(t, err)

	swapState := quote.SwapInfo.SwapStateAfter.(*BoostedFeesPoolSwapState)
	require.Equal(t, uint64(100), swapState.LastExecutionTime)
	require.True(t, swapState.Token0Rate.IsZero())
	require.True(t, swapState.Token1Rate.IsZero())
}

func TestBoostedFeesPoolQuoteTracksTimeWithoutDeltas(t *testing.T) {
	t.Parallel()

	pool := testBoostedFeesPool(100, 0, 0, nil)
	quote, err := pool.quoteWithTimestampFn(uint256.NewInt(0), false, func() uint64 { return 150 })
	require.NoError(t, err)

	swapState := quote.SwapInfo.SwapStateAfter.(*BoostedFeesPoolSwapState)
	require.Equal(t, uint64(150), swapState.LastExecutionTime)
	require.True(t, swapState.Token0Rate.IsZero())
	require.True(t, swapState.Token1Rate.IsZero())
}

func TestBoostedFeesPoolQuoteAppliesDeltas(t *testing.T) {
	t.Parallel()

	rate := uint64(1 << 32)
	pool := testBoostedFeesPool(
		0,
		rate,
		0,
		[]TimeRateDelta{
			{
				Time:   200,
				Delta0: int256.NewInt(-int64(rate)),
				Delta1: int256.NewInt(0),
			},
		},
	)

	quote, err := pool.quoteWithTimestampFn(uint256.NewInt(0), false, func() uint64 { return 300 })
	require.NoError(t, err)

	swapState := quote.SwapInfo.SwapStateAfter.(*BoostedFeesPoolSwapState)
	require.Equal(t, uint64(300), swapState.LastExecutionTime)
	require.True(t, swapState.Token0Rate.IsZero())
	require.True(t, swapState.Token1Rate.IsZero())
}

func TestBoostedFeesPoolQuoteIgnoresFutureDeltas(t *testing.T) {
	t.Parallel()

	rate := uint64(1 << 32)
	pool := testBoostedFeesPool(
		100,
		0,
		0,
		[]TimeRateDelta{
			{
				Time:   200,
				Delta0: int256.NewInt(int64(rate)),
				Delta1: int256.NewInt(0),
			},
			{
				Time:   300,
				Delta0: int256.NewInt(-int64(rate)),
				Delta1: int256.NewInt(0),
			},
		},
	)

	quote, err := pool.quoteWithTimestampFn(uint256.NewInt(0), false, func() uint64 { return 150 })
	require.NoError(t, err)

	swapState := quote.SwapInfo.SwapStateAfter.(*BoostedFeesPoolSwapState)
	require.Equal(t, uint64(150), swapState.LastExecutionTime)
	require.True(t, swapState.Token0Rate.IsZero())
	require.True(t, swapState.Token1Rate.IsZero())
}

func testBoostedFeesPool(lastDonateTime, donateRate0, donateRate1 uint64, deltas []TimeRateDelta) *BoostedFeesPool {
	liquidity := uint256.NewInt(1_000_000)
	state := NewBoostedFeesPoolState(
		NewConcentratedPoolState(
			NewConcentratedPoolSwapState(
				ekubomath.ToSqrtRatio(0),
				liquidity,
				0,
			),
			[]Tick{
				{
					Number:         -10,
					LiquidityDelta: big256.SInt256(liquidity),
				},
				{
					Number:         10,
					LiquidityDelta: big256.SNeg(liquidity),
				},
			},
			[2]int32{ekubomath.MinTick, ekubomath.MaxTick},
			0,
		),
		NewTimedPoolState(
			NewTimedPoolSwapState(
				uint256.NewInt(donateRate0),
				uint256.NewInt(donateRate1),
				lastDonateTime,
			),
			deltas,
		),
	)

	return NewBoostedFeesPool(
		NewPoolKey(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			common.HexToAddress("0x0000000000000000000000000000000000000001"),
			NewPoolConfig(common.Address{}, 0, NewConcentratedPoolTypeConfig(1)),
		),
		state,
	)
}
