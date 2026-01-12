package pools

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func mevCapturePoolKey(fee uint64, tickSpacing uint32) *ConcentratedPoolKey {
	return NewPoolKey(common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		NewPoolConfig(common.HexToAddress("0x0000000000000000000000000000000000000002"), fee, NewConcentratedPoolTypeConfig(tickSpacing)))
}

func TestSwapInputAmountToken0(t *testing.T) {
	t.Parallel()

	liquidity := uint256.NewInt(28_898_102)
	fee := new(uint256.Int).Div(new(uint256.Int).Lsh(big256.U1, 64), uint256.NewInt(100)).Uint64()

	pool := NewMevCapturePool(mevCapturePoolKey(fee, 20_000), &BasePoolState{
		BasePoolSwapState: &BasePoolSwapState{
			SqrtRatio:       math.ToSqrtRatio(700_000),
			Liquidity:       new(uint256.Int).Set(liquidity),
			ActiveTickIndex: 0,
		},
		SortedTicks: []Tick{
			{
				Number:         600_000,
				LiquidityDelta: big256.SInt256(liquidity),
			},
			{
				Number:         800_000,
				LiquidityDelta: big256.SNeg(liquidity),
			},
		},
		TickBounds: [2]int32{math.MinTick, math.MaxTick},
		ActiveTick: 700_000,
	})

	quote, err := pool.Quote(uint256.NewInt(100_000), false)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(100_000), quote.ConsumedAmount)
	require.Equal(t, uint256.NewInt(197_432), quote.CalculatedAmount)

	quote, err = pool.Quote(uint256.NewInt(300_000), false)
	require.NoError(t, err)

	pool.SetSwapState(quote.SwapInfo.SwapStateAfter)

	quote, err = pool.Quote(uint256.NewInt(300_000), false)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(300_000), quote.ConsumedAmount)
	require.Equal(t, uint256.NewInt(556_308), quote.CalculatedAmount)
}

func TestSwapOutputAmountToken0(t *testing.T) {
	t.Parallel()

	liquidity := uint256.NewInt(28_898_102)
	fee := new(uint256.Int).Div(new(uint256.Int).Lsh(big256.U1, 64), uint256.NewInt(100)).Uint64()

	pool := NewMevCapturePool(mevCapturePoolKey(fee, 20_000), &BasePoolState{
		BasePoolSwapState: &BasePoolSwapState{
			SqrtRatio:       math.ToSqrtRatio(700_000),
			Liquidity:       new(uint256.Int).Set(liquidity),
			ActiveTickIndex: 0,
		},
		SortedTicks: []Tick{
			{
				Number:         600_000,
				LiquidityDelta: big256.SInt256(liquidity),
			},
			{
				Number:         800_000,
				LiquidityDelta: big256.SNeg(liquidity),
			},
		},
		TickBounds: [2]int32{math.MinTick, math.MaxTick},
		ActiveTick: 700_000,
	})

	quote, err := pool.Quote(new(uint256.Int).Neg(uint256.NewInt(100_000)), false)
	require.NoError(t, err)

	require.Equal(t, new(uint256.Int).Neg(uint256.NewInt(100_000)), quote.ConsumedAmount)
	require.Equal(t, uint256.NewInt(205_416), quote.CalculatedAmount)
}
