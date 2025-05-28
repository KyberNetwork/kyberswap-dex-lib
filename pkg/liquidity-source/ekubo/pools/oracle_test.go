package pools

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
)

var oraclePoolKey = &PoolKey{
	Token0: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	Token1: common.HexToAddress("0x0000000000000000000000000000000000000001"),
	Config: PoolConfig{
		Fee:         0,
		TickSpacing: 0,
		Extension:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
	},
}

func TestQuoteToken1Input(t *testing.T) {
	t.Parallel()
	p := NewOraclePool(
		oraclePoolKey,
		&OraclePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: math.TwoPow128,
			},
			Liquidity: big.NewInt(1_000_000_000),
		},
	)

	quote, err := p.Quote(big.NewInt(1000), true)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(999)))
	require.Zero(t, quote.ConsumedAmount.Cmp(big.NewInt(1000)))
}

func TestQuoteToken0Input(t *testing.T) {
	t.Parallel()
	p := NewOraclePool(
		oraclePoolKey,
		&OraclePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: math.TwoPow128,
			},
			Liquidity: big.NewInt(1_000_000_000),
		},
	)

	quote, err := p.Quote(big.NewInt(1000), false)
	require.NoError(t, err)

	require.Zero(t, quote.CalculatedAmount.Cmp(big.NewInt(999)))
	require.Zero(t, quote.ConsumedAmount.Cmp(big.NewInt(1000)))
}
