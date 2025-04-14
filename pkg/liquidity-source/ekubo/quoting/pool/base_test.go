package pool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var maxTickBounds = [2]int32{math.MinTick, math.MaxTick}

func ticks(liquidity *big.Int) []quoting.Tick {
	return []quoting.Tick{
		{Number: math.MinTick, LiquidityDelta: new(big.Int).Set(liquidity)},
		{Number: math.MaxTick, LiquidityDelta: new(big.Int).Set(liquidity)},
	}
}

func poolKey(tickSpacing uint32, fee uint64) *quoting.PoolKey {
	return quoting.NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		quoting.Config{
			Fee:         fee,
			TickSpacing: tickSpacing,
			Extension:   common.Address{},
		},
	)
}

func TestQuoteZeroLiquidityToken1Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting.NewPoolState(
			new(big.Int),
			math.TwoPow128,
			0,
			ticks(new(big.Int)),
			maxTickBounds,
		),
	)
	_, err := p.Quote(bignum.One, true)
	require.ErrorIs(t, err, pool.ErrZeroAmount)
}

func TestQuoteZeroLiquidityToken0Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting.NewPoolState(
			new(big.Int),
			math.TwoPow128,
			0,
			ticks(new(big.Int)),
			maxTickBounds,
		),
	)
	_, err := p.Quote(bignum.One, true)
	require.ErrorIs(t, err, pool.ErrZeroAmount)
}

func TestQuoteLiquidityToken1Input(t *testing.T) {
	p := pool.NewBasePool(
		poolKey(1, 0),
		quoting.NewPoolState(
			bignum.NewBig("1_000_000_000"),
			math.TwoPow128,
			0,
			[]quoting.Tick{
				{Number: 0, LiquidityDelta: bignum.NewBig("1_000_000_000")},
				{Number: 1, LiquidityDelta: bignum.NewBig("-1_000_000_000")},
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
		quoting.NewPoolState(
			new(big.Int),
			math.ToSqrtRatio(1),
			1,
			[]quoting.Tick{
				{Number: 0, LiquidityDelta: bignum.NewBig("1_000_000_000")},
				{Number: 1, LiquidityDelta: bignum.NewBig("-1_000_000_000")},
			},
			[2]int32{0, 1},
		),
	)

	quote, err := p.Quote(big.NewInt(1000), false)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(499)))
}
