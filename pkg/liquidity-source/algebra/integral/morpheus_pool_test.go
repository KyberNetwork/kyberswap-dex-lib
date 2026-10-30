package integral

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Snapshot of Morpheus WETH/HOOD pool 0xcd9e…13d3 on Robinhood (captured 2026-10-13).
// Uses LastFee only (pluginV2 off) so the test stays offline and deterministic.
const morpheusWethHoodPoolJSON = `{
  "address":"0xcd9e25506cd72db14286165dd940ccfc032613d3",
  "exchange":"morpheus",
  "type":"algebra-integral",
  "timestamp":1760366400,
  "reserves":["511401885294402536","9749995247470463052966376739"],
  "tokens":[
    {"address":"0x0bd7d308f8e1639fab988df18a8011f41eacad73","symbol":"WETH","decimals":18,"swappable":true},
    {"address":"0xd25a41bb6174e519f9c70e12bf15d6bad5af7377","symbol":"HOOD","decimals":18,"swappable":true}
  ],
  "extra":"{\"liq\":\"446570136036924783687465\",\"gS\":{\"price\":\"1729793700123338579575419555749156\",\"tick\":199833,\"lF\":500,\"pC\":215,\"cF\":100,\"un\":true},\"ticks\":[{\"Index\":-887160,\"LiquidityGross\":\"446570136036924783687465\",\"LiquidityNet\":\"446570136036924783687465\"},{\"Index\":200340,\"LiquidityGross\":\"446570136036924783687465\",\"LiquidityNet\":\"-446570136036924783687465\"}],\"tS\":60}",
  "staticExtra":"{}"
}`

func TestMorpheusConfig(t *testing.T) {
	cfg := MorpheusConfig()
	require.Equal(t, valueobject.ExchangeMorpheus, cfg.DexID)
	require.Equal(t, MorpheusAnalyticsSubgraph, cfg.SubgraphAPI)
	require.Equal(t, MorpheusTickLens, cfg.TickLensAddress)
	require.True(t, cfg.UseBasePluginV2)
	require.Equal(t, "0x10253594A832f967994b44f33411940533302ACb", MorpheusFactory)
}

func TestMorpheusWethHoodSimulator(t *testing.T) {
	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(morpheusWethHoodPoolJSON), &entityPool))

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	require.Equal(t, valueobject.ExchangeMorpheus, sim.GetExchange())
	require.Equal(t, DexType, sim.GetType())

	weth := entityPool.Tokens[0].Address
	hood := entityPool.Tokens[1].Address

	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1e15)},
		TokenOut:      hood,
	})
	require.NoError(t, err)
	require.NotNil(t, out.TokenAmountOut)
	require.True(t, out.TokenAmountOut.Amount.Sign() > 0)
	require.True(t, out.RemainingTokenAmountIn.Amount.Sign() == 0)

	out2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: hood, Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)},
		TokenOut:      weth,
	})
	require.NoError(t, err)
	require.True(t, out2.TokenAmountOut.Amount.Sign() > 0)
}
