package uniswapv2

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The registered uniswap-v2 factory is legacy (big.Int). FactoryPoolSim must wrap it, and its uint256 quote must equal
// the legacy big.Int quote for the same input. A mismatch would mean the adapter changed a route.
func TestFactoryUMatchesLegacyQuote(t *testing.T) {
	t.Parallel()
	u, err := pool.FactoryPoolSim(DexType)(pool.FactoryParams{EntityPool: poolEntity})
	require.NoError(t, err)

	tokenIn := poolSim.Info.Tokens[0]
	tokenOut := poolSim.Info.Tokens[1]
	for _, amt := range []int64{1e6, 1e9, 1e12} {
		legacy, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(amt)},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)

		native, err := u.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *uint256.NewInt(uint64(amt))},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)
		require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountIn %d", amt)
	}
}
