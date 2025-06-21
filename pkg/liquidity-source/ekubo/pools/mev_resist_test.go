package pools

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func mevResistPoolKey(fee uint64, tickSpacing uint32) *PoolKey {
	return NewPoolKey(common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		PoolConfig{
			Fee:         fee,
			TickSpacing: tickSpacing,
			Extension:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
		})
}

func TestSwapInputAmountToken0(t *testing.T) {
	t.Parallel()

	liquidity := big.NewInt(28_898_102)
	fee := new(big.Int).Div(new(big.Int).Lsh(bignum.One, 64), big.NewInt(100)).Uint64()

	pool := NewMevResistPool(mevResistPoolKey(fee, 20_000), &BasePoolState{
		BasePoolSwapState: &BasePoolSwapState{
			SqrtRatio:       math.ToSqrtRatio(700_000),
			Liquidity:       new(big.Int).Set(liquidity),
			ActiveTickIndex: 0,
		},
		SortedTicks: []Tick{
			{
				Number:         600_000,
				LiquidityDelta: new(big.Int).Set(liquidity),
			},
			{
				Number:         800_000,
				LiquidityDelta: new(big.Int).Neg(liquidity),
			},
		},
		TickBounds: [2]int32{math.MinTick, math.MaxTick},
		ActiveTick: 700_000,
	})

	quote, err := pool.Quote(big.NewInt(100_000), false)
	require.NoError(t, err)

	require.Equal(t, big.NewInt(100_000), quote.ConsumedAmount)
	require.Equal(t, big.NewInt(197_432), quote.CalculatedAmount)

	quote, err = pool.Quote(big.NewInt(300_000), false)
	require.NoError(t, err)

	pool.SetSwapState(quote.SwapInfo.SwapStateAfter)

	quote, err = pool.Quote(big.NewInt(300_000), false)
	require.NoError(t, err)

	require.Equal(t, big.NewInt(300_000), quote.ConsumedAmount)
	require.Equal(t, big.NewInt(556_308), quote.CalculatedAmount)
}

func TestSwapOutputAmountToken0(t *testing.T) {
	t.Parallel()

	liquidity := big.NewInt(28_898_102)
	fee := new(big.Int).Div(new(big.Int).Lsh(bignum.One, 64), big.NewInt(100)).Uint64()

	pool := NewMevResistPool(mevResistPoolKey(fee, 20_000), &BasePoolState{
		BasePoolSwapState: &BasePoolSwapState{
			SqrtRatio:       math.ToSqrtRatio(700_000),
			Liquidity:       new(big.Int).Set(liquidity),
			ActiveTickIndex: 0,
		},
		SortedTicks: []Tick{
			{
				Number:         600_000,
				LiquidityDelta: new(big.Int).Set(liquidity),
			},
			{
				Number:         800_000,
				LiquidityDelta: new(big.Int).Neg(liquidity),
			},
		},
		TickBounds: [2]int32{math.MinTick, math.MaxTick},
		ActiveTick: 700_000,
	})

	quote, err := pool.Quote(big.NewInt(-100_000), false)
	require.NoError(t, err)

	require.Equal(t, big.NewInt(-100_000), quote.ConsumedAmount)
	require.Equal(t, big.NewInt(205_416), quote.CalculatedAmount)
}
