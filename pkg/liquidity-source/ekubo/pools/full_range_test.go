package pools

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestFullRangePoolQuote(t *testing.T) {
	t.Parallel()
	poolKey := func(fee uint64) *PoolKey {
		return NewPoolKey(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			common.HexToAddress("0x0000000000000000000000000000000000000001"),
			PoolConfig{
				Fee:         fee,
				TickSpacing: 0,
				Extension:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
			},
		)
	}

	t.Run("zero_liquidity", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), &FullRangePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: math.TwoPow128,
			},
			Liquidity: new(big.Int),
		})

		quote, err := pool.Quote(big.NewInt(1_000), false)
		require.NoError(t, err)

		require.Equal(t, bignum.ZeroBI, quote.CalculatedAmount)
	})

	t.Run("with_liqudity_token0_input", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), &FullRangePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: math.TwoPow128,
			},
			Liquidity: big.NewInt(1_000_000),
		})

		quote, err := pool.Quote(big.NewInt(1_000), false)
		require.NoError(t, err)

		require.Equal(t, big.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("with_liqudity_token1_input", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), &FullRangePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: math.TwoPow128,
			},
			Liquidity: big.NewInt(1_000_000),
		})

		quote, err := pool.Quote(big.NewInt(1_000), true)
		require.NoError(t, err)

		require.Equal(t, big.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("with_fee", func(t *testing.T) {
		pool := NewFullRangePool(
			poolKey(1<<32), // 0.01% fee
			&FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.TwoPow128,
				},
				Liquidity: big.NewInt(1_000_000),
			},
		)

		quote, err := pool.Quote(big.NewInt(1_000), false)
		require.NoError(t, err)

		require.Positive(t, quote.FeesPaid.Sign())
	})
}
