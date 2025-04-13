package pool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
)

func oraclePoolKey() *quoting.PoolKey {
	return quoting.NewPoolKey(
		common.HexToAddress("0x0000000000000000000000000000000000000000"),
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		quoting.Config{
			Fee:         0,
			TickSpacing: 0,
			Extension:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
		},
	)
}

func TestQuoteToken1Input(t *testing.T) {
	p := pool.NewOraclePool(
		oraclePoolKey(),
		quoting.NewPoolState(
			big.NewInt(1_000_000_000),
			math.ToSqrtRatio(0),
			0,
			ticks(big.NewInt(1_000_000_000)),
			maxTickBounds,
		),
	)

	quote, err := p.Quote(big.NewInt(1000), true)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(999)))
	require.Zero(t, quote.ConsumedAmount.Cmp(big.NewInt(1000)))
}

func TestQuoteToken0Input(t *testing.T) {
	p := pool.NewOraclePool(
		oraclePoolKey(),
		quoting.NewPoolState(
			big.NewInt(1_000_000_000),
			math.ToSqrtRatio(0),
			0,
			ticks(big.NewInt(1_000_000_000)),
			maxTickBounds,
		),
	)

	quote, err := p.Quote(big.NewInt(1000), false)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(999)))
	require.Zero(t, quote.ConsumedAmount.Cmp(big.NewInt(1000)))
}
