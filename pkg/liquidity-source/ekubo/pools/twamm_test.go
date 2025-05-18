package pools

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestTwammPoolQuote(t *testing.T) {
	t.Parallel()
	poolKey := NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		PoolConfig{
			Fee:         0,
			TickSpacing: 0,
			Extension:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
		},
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
				Liquidity: big.NewInt(1_000_000_000),
			},
			Token0SaleRate:     bignum.ZeroBI,
			Token1SaleRate:     bignum.ZeroBI,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rates_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     bignum.ZeroBI,
			Token1SaleRate:     bignum.ZeroBI,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(990), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token0_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:     math.TwoPow32,
			Token1SaleRate:     bignum.ZeroBI,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(999), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token1_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:     bignum.ZeroBI,
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(998), quote.CalculatedAmount)
	})

	t.Run("non_zero_sale_rate_token1_max_price_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:     bignum.ZeroBI,
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, bignum.ZeroBI, quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token0_at_max_price_deltas_move_price_down_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:    bignum.ZeroBI,
			Token1SaleRate:    math.TwoPow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: new(big.Int).Mul(big.NewInt(100_000), math.TwoPow32),
					SaleRateDelta1: bignum.ZeroBI,
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(2555), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token1_close_at_min_price_deltas_move_price_up_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MinSqrtRatio,
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:    math.TwoPow32,
			Token1SaleRate:    bignum.ZeroBI,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: bignum.ZeroBI,
					SaleRateDelta1: new(big.Int).Mul(big.NewInt(100_000), math.TwoPow32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(390), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token0_at_max_price_deltas_move_price_down_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MaxSqrtRatio,
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:    bignum.ZeroBI,
			Token1SaleRate:    math.TwoPow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: new(big.Int).Mul(big.NewInt(100_000), math.TwoPow32),
					SaleRateDelta1: bignum.ZeroBI,
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(390), quote.CalculatedAmount)
	})

	t.Run("zero_sale_rate_token1_at_min_price_deltas_move_price_up_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.MinSqrtRatio,
				},
				Liquidity: big.NewInt(1_000_000),
			},
			Token0SaleRate:    math.TwoPow32,
			Token1SaleRate:    bignum.ZeroBI,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: bignum.ZeroBI,
					SaleRateDelta1: new(big.Int).Mul(big.NewInt(100_000), math.TwoPow32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(2555), quote.CalculatedAmount)
	})

	t.Run("one_e18_sale_rates_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     math.TwoPow32,
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(990), quote.CalculatedAmount)
	})

	t.Run("one_e18_sale_rates_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     math.TwoPow32,
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("token0_sale_rate_greater_than_token1_sale_rate_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(1_000),
			},
			Token0SaleRate:     new(big.Int).Lsh(bignum.Ten, 32),
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(717), quote.CalculatedAmount)
	})

	t.Run("token1_sale_rate_greater_than_token0_sale_rate_no_sale_rate_deltas_quote_token1", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     math.TwoPow32,
			Token1SaleRate:     new(big.Int).Lsh(bignum.Ten, 32),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), true, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(984), quote.CalculatedAmount)
	})

	t.Run("token0_sale_rate_greater_than_token1_sale_rate_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     new(big.Int).Lsh(bignum.Ten, 32),
			Token1SaleRate:     math.TwoPow32,
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(983), quote.CalculatedAmount)
	})

	t.Run("token1_sale_rate_greater_than_token0_sale_rate_no_sale_rate_deltas_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:     math.TwoPow32,
			Token1SaleRate:     new(big.Int).Lsh(bignum.Ten, 32),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(994), quote.CalculatedAmount)
	})

	t.Run("sale_rate_deltas_goes_to_zero_halfway_through_execution_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:    math.TwoPow32,
			Token1SaleRate:    math.TwoPow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: new(big.Int).Neg(math.TwoPow32),
					SaleRateDelta1: new(big.Int).Neg(math.TwoPow32),
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("sale_rate_deltas_doubles_halfway_through_execution_quote_token0", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(1),
				},
				Liquidity: big.NewInt(100_000),
			},
			Token0SaleRate:    math.TwoPow32,
			Token1SaleRate:    math.TwoPow32,
			LastExecutionTime: 0,
			VirtualOrderDeltas: []TwammSaleRateDelta{
				{
					Time:           16,
					SaleRateDelta0: math.TwoPow32,
					SaleRateDelta1: math.TwoPow32,
				},
			},
		})

		quote, err := pool.quoteWithTimestampFn(big.NewInt(1000), false, fixedTimestampFn(32))
		require.NoError(t, err)

		require.Equal(t, big.NewInt(989), quote.CalculatedAmount)
	})

	t.Run("compare_to_contract_output", func(t *testing.T) {
		pool := NewTwammPool(poolKey, &TwammPoolState{
			FullRangePoolState: &FullRangePoolState{
				FullRangePoolSwapState: &FullRangePoolSwapState{
					SqrtRatio: math.ToSqrtRatio(693147),
				},
				Liquidity: bignum.NewBig("70_710_696_755_630_728_101_718_334"),
			},
			Token0SaleRate:     bignum.NewBig("10_526_880_627_450_980_392_156_862_745"),
			Token1SaleRate:     bignum.NewBig("10_526_880_627_450_980_392_156_862_745"),
			LastExecutionTime:  0,
			VirtualOrderDeltas: []TwammSaleRateDelta{},
		})

		tenPow18 := new(big.Int).Exp(bignum.Ten, big.NewInt(18), nil)

		quote, err := pool.quoteWithTimestampFn(tenPow18.Mul(tenPow18, big.NewInt(10_000)), false, fixedTimestampFn(2_040))
		require.NoError(t, err)

		require.Equal(t, bignum.NewBig("19993991114278789946056"), quote.CalculatedAmount)
	})
}
