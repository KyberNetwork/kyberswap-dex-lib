package pools

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestTwammPoolQuote(t *testing.T) {
	t.Parallel()
	poolKey := NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		NewPoolConfig(common.HexToAddress("0x0000000000000000000000000000000000000002"), 0, NewFullRangePoolTypeConfig()),
	)

	fixedTimestampFn := func(timestamp uint64) func() uint64 {
		return func() uint64 { return timestamp }
	}

	t.Run("zero_sale_rates_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(1_000_000_000),
			},
			Token0SaleRate:     big256.U0,
			Token1SaleRate:     big256.U0,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rates_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     big256.U0,
			Token1SaleRate:     big256.U0,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(990), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token0_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:     big256.U2Pow32,
			Token1SaleRate:     big256.U0,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token1_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:     big256.U0,
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(998), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token1_max_price_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:     big256.U0,
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big256.U0, quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token0_at_max_price_deltas_move_price_down_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:    big256.U0,
			Token1SaleRate:    big256.U2Pow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(1e5 << 32),
					SaleRateDelta1: int256.NewInt(0),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(2555), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token1_close_at_min_price_deltas_move_price_up_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MinSqrtRatio,
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:    big256.U2Pow32,
			Token1SaleRate:    big256.U0,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(0),
					SaleRateDelta1: int256.NewInt(1e5 << 32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(390), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token0_at_max_price_deltas_move_price_down_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:    big256.U0,
			Token1SaleRate:    big256.U2Pow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(1e5 << 32),
					SaleRateDelta1: int256.NewInt(0),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(390), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token1_at_min_price_deltas_move_price_up_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MinSqrtRatio,
				},
				Liquidity: uint256.NewInt(1_000_000),
			},
			Token0SaleRate:    big256.U2Pow32,
			Token1SaleRate:    big256.U0,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(0),
					SaleRateDelta1: int256.NewInt(1e5 << 32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(2555), quote.CalculatedAmount)
	})

	t.Run("one_e18_sale_rates_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     big256.U2Pow32,
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(990), quote.CalculatedAmount)
	})

	t.Run("one_e18_sale_rates_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     big256.U2Pow32,
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("token0_sale_rate_greater_than_token1_sale_rate_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(1_000),
			},
			Token0SaleRate:     new(uint256.Int).Lsh(big256.U10, 32),
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(717), quote.CalculatedAmount)
	})

	t.Run("token1_sale_rate_greater_than_token0_sale_rate_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     big256.U2Pow32,
			Token1SaleRate:     new(uint256.Int).Lsh(big256.U10, 32),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(984), quote.CalculatedAmount)
	})

	t.Run("token0_sale_rate_greater_than_token1_sale_rate_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     new(uint256.Int).Lsh(big256.U10, 32),
			Token1SaleRate:     big256.U2Pow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(983), quote.CalculatedAmount)
	})

	t.Run("token1_sale_rate_greater_than_token0_sale_rate_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:     big256.U2Pow32,
			Token1SaleRate:     new(uint256.Int).Lsh(big256.U10, 32),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(994), quote.CalculatedAmount)
	})

	t.Run("sale_rate_deltas_goes_to_zero_halfway_through_execution_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:    big256.U2Pow32,
			Token1SaleRate:    big256.U2Pow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(-1 << 32),
					SaleRateDelta1: int256.NewInt(1 << 32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("sale_rate_deltas_doubles_halfway_through_execution_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: uint256.NewInt(100_000),
			},
			Token0SaleRate:    big256.U2Pow32,
			Token1SaleRate:    big256.U2Pow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: int256.NewInt(1 << 32),
					SaleRateDelta1: int256.NewInt(1 << 32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(uint256.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, uint256.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("compare_to_contract_output", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(693147),
				},
				Liquidity: big256.New("70710696755630728101718334"),
			},
			Token0SaleRate:     big256.New("10526880627450980392156862745"),
			Token1SaleRate:     big256.New("10526880627450980392156862745"),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		tenPow18 := new(uint256.Int).Set(big256.BONE)

		quote, err := pool.quoteWithTimestampFn(tenPow18.Mul(tenPow18, uint256.NewInt(10_000)), false, fixedTimestampFn(2_040))
		require.NoError(t, err)

		require.Equal(t, big256.New("19993991114278789946056"), quote.CalculatedAmount)
	})
}
