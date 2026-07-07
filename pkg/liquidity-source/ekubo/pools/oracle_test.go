package pools

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
				SqrtRatio: big256.U2Pow128,
			},
			Liquidity: uint256.NewInt(1_000_000_000),
		},
	)

	quote, err := p.Quote(uint256.NewInt(1000), true)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	require.Equal(t, uint256.NewInt(1000), quote.ConsumedAmount)
}

func TestQuoteToken0Input(t *testing.T) {
	t.Parallel()
	p := NewOraclePool(
		oraclePoolKey,
		&OraclePoolState{
			FullRangePoolSwapState: &FullRangePoolSwapState{
				SqrtRatio: big256.U2Pow128,
			},
			Liquidity: uint256.NewInt(1_000_000_000),
		},
	)

	quote, err := p.Quote(uint256.NewInt(1000), false)
	require.NoError(t, err)

	require.Equal(t, uint256.NewInt(999), quote.CalculatedAmount)
	require.Equal(t, uint256.NewInt(1000), quote.ConsumedAmount)
}
