package msgpack

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	flywheel "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/flywheel-fun"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestFlywheelCompositeSerialization(t *testing.T) {
	b, err := os.ReadFile("../liquidity-source/flywheel-fun/testdata/fork-markets.json")
	require.NoError(t, err)
	var markets []entity.Pool
	require.NoError(t, json.Unmarshal(b, &markets))
	for _, p := range markets {
		before, err := flywheel.NewPoolSimulator(p)
		require.NoError(t, err)
		encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: before})
		require.NoError(t, err)
		decoded, err := DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: flywheel.WETH, Amount: big.NewInt(1000000000)}, TokenOut: p.Address}
		expected, err := before.CalcAmountOut(params)
		require.NoError(t, err)
		actual, err := decoded[p.Address].CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, expected.TokenAmountOut, actual.TokenAmountOut)
		require.Equal(t, expected.RemainingTokenAmountIn, actual.RemainingTokenAmountIn)
		decoded[p.Address].UpdateBalance(pool.UpdateBalanceParams{SwapInfo: actual.SwapInfo})
		fresh, err := before.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, expected.TokenAmountOut, fresh.TokenAmountOut)
	}
}
