package pools

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestFullRangePoolQuote(t *testing.T) {
	t.Parallel()
	poolKey := func(fee uint64) *FullRangePoolKey {
		return NewPoolKey(
			common.HexToAddress("0x0000000000000000000000000000000000000000"),
			common.HexToAddress("0x0000000000000000000000000000000000000001"),
			NewPoolConfig(common.HexToAddress("0x0000000000000000000000000000000000000002"), fee, NewFullRangePoolTypeConfig()),
		)
	}

	t.Run("zero_liquidity", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), NewFullRangePoolState(
			NewFullRangePoolSwapState(big256.U2Pow128),
			new(uint256.Int),
		))

		quote, err := pool.Quote(uint256.NewInt(1_000), false)
		require.NoError(t, err)

		require.Equal(t, big256.U0, quote.CalculatedAmount)
	})

	t.Run("with_liqudity_token0_input", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), NewFullRangePoolState(
			NewFullRangePoolSwapState(big256.U2Pow128),
			uint256.NewInt(1_000_000),
		))

		quote, err := pool.Quote(uint256.NewInt(1_000), false)
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("with_liqudity_token1_input", func(t *testing.T) {
		pool := NewFullRangePool(poolKey(0), NewFullRangePoolState(
			NewFullRangePoolSwapState(big256.U2Pow128),
			uint256.NewInt(1_000_000),
		))

		quote, err := pool.Quote(uint256.NewInt(1_000), true)
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("with_fee", func(t *testing.T) {
		pool := NewFullRangePool(
			poolKey(1<<32), // 0.01% fee
			NewFullRangePoolState(
				NewFullRangePoolSwapState(big256.U2Pow128),
				uint256.NewInt(1_000_000),
			),
		)

		quote, err := pool.Quote(uint256.NewInt(1_000), false)
		require.NoError(t, err)

		require.Positive(t, quote.FeesPaid.Sign())
	})
}
