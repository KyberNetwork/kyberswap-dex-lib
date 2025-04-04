package pool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	math2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	quoting2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
)

var maxTickBounds = [2]int32{math2.MinTick, math2.MaxTick}

func ticks(liquidity *big.Int) []quoting2.Tick {
	return []quoting2.Tick{
		{
			Number:         math2.MinTick,
			LiquidityDelta: new(big.Int).Set(liquidity),
		},
		{
			Number:         math2.MaxTick,
			LiquidityDelta: new(big.Int).Set(liquidity),
		},
	}
}

func poolKey(tickSpacing uint32, fee uint64) quoting2.PoolKey {
	return quoting2.NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		quoting2.Config{
			Fee:         fee,
			TickSpacing: tickSpacing,
			Extension:   common.Address{},
		},
	)
}

func TestQuoteZeroLiquidityToken1Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting2.NewPoolState(
			new(big.Int),
			math2.TwoPow128,
			0,
			ticks(new(big.Int)),
			maxTickBounds,
		),
	)

	quote, err := p.Quote(math2.One, true)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Sign())
}

func TestQuoteZeroLiquidityToken0Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting2.NewPoolState(
			new(big.Int),
			math2.TwoPow128,
			0,
			ticks(new(big.Int)),
			maxTickBounds,
		),
	)

	quote, err := p.Quote(math2.One, false)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Sign())
}

func TestQuoteLiquidityToken1Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting2.NewPoolState(
			math2.IntFromString("1_000_000_000"),
			math2.TwoPow128,
			0,
			[]quoting2.Tick{
				{
					Number:         0,
					LiquidityDelta: math2.IntFromString("1_000_000_000"),
				},
				{
					Number:         1,
					LiquidityDelta: math2.IntFromString("-1_000_000_000"),
				},
			},
			[2]int32{0, 1},
		),
	)

	quote, err := p.Quote(big.NewInt(1000), true)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(499)))
}

func TestQuoteLiquidityToken0Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting2.NewPoolState(
			new(big.Int),
			math2.ToSqrtRatio(1),
			1,
			[]quoting2.Tick{
				{
					Number:         0,
					LiquidityDelta: math2.IntFromString("1_000_000_000"),
				},
				{
					Number:         1,
					LiquidityDelta: math2.IntFromString("-1_000_000_000"),
				},
			},
			[2]int32{0, 1},
		),
	)

	quote, err := p.Quote(big.NewInt(1000), false)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(499)))
}
