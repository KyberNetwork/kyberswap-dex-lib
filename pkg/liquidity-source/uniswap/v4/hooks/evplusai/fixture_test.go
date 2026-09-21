package evplusai

import (
	_ "embed"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

//go:embed testdata/quotes.json
var quotesFixture []byte

func TestDeployedQuoteFixtures(t *testing.T) {
	t.Parallel()
	var f fixture
	require.NoError(t, json.Unmarshal(quotesFixture, &f))
	require.Len(t, f.Quotes, 40)
	for _, q := range f.Quotes {
		t.Run(q.Name, func(t *testing.T) { t.Parallel(); compareQuote(t, f, q) })
	}
}

func TestNativeRouteMetadata(t *testing.T) {
	t.Parallel()
	var f fixture
	require.NoError(t, json.Unmarshal(quotesFixture, &f))
	var extra uniswapv4.Extra
	require.NoError(t, json.Unmarshal([]byte(f.Pool.Extra), &extra))
	extra.HookExtra = f.Quotes[0].Extra
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	f.Pool.Extra = string(raw)
	sim, err := uniswapv4.NewPoolSimulator(f.Pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	require.False(t, sim.V3Pool.ExactTickTraversal)
	meta := sim.GetMetaInfo(weth, usdg).(uniswapv4.PoolMetaInfo)
	require.Equal(t, common.Address{}, meta.TokenIn)
	require.Equal(t, common.HexToAddress(usdg), meta.TokenOut)
	require.Equal(t, HookAddress, meta.HookAddress)
	require.Empty(t, meta.HookData)
	require.Equal(t, f.Pool.BlockNumber, meta.BlockNumber)
	reverse := sim.GetMetaInfo(usdg, weth).(uniswapv4.PoolMetaInfo)
	require.Equal(t, common.Address{}, reverse.TokenOut)
	require.EqualValues(t, 8388608, meta.Fee)
	require.EqualValues(t, 10, meta.TickSpacing)
	// Updating a clone must not mutate the original pool price or fee snapshot.
	clone := sim.CloneState().(*uniswapv4.PoolSimulator)
	quote, err := clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1000000000000000)}, TokenOut: usdg})
	require.NoError(t, err)
	originalPrice := sim.V3Pool.SqrtRatioX96
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1000000000000000)}, TokenAmountOut: *quote.TokenAmountOut, SwapInfo: quote.SwapInfo})
	require.Equal(t, originalPrice, sim.V3Pool.SqrtRatioX96)
	require.NotEqual(t, originalPrice, clone.V3Pool.SqrtRatioX96)
}
