package everlongcollvault

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func katanaTestConfig() *Config {
	return &Config{
		DexID:      DexType,
		ChainID:    valueobject.ChainIDKatana,
		Rebalancer: "0x4eee1C828B6cAFb8CC7Bcf44D05F83483e499b23",
		Stable:     "0x0F26bBb8962d73bC891327F14dB5162D5279899F",
		Volatile:   "0x0913DA6Da4b42f538B445599b46Bb4622342Cf52",
	}
}

func katanaRPCClient() *ethrpc.Client {
	client := ethrpc.New("https://rpc.katana.network")
	client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	return client
}

// TestLiveListTrackQuote runs the full lister -> tracker -> simulator pipeline against
// live Katana: resolve the contract graph on-chain, refresh the vault snapshot through
// the tracker's multicall, and quote both directions off the refreshed entity.
func TestLiveListTrackQuote(t *testing.T) {
	test.SkipCI(t)

	cfg := katanaTestConfig()
	client := katanaRPCClient()

	lister := NewPoolsListUpdater(cfg, client)
	pools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(pools[0].StaticExtra), &staticExtra))
	require.Equal(t, "0x985a6b410f7abe294e4b0fa938d3e8d2f83e79d1", staticExtra.Swapper)
	require.Equal(t, "0x3f7da0ade05242d389a86abaf1ba2a85e0563a86", staticExtra.CollVault)
	require.Equal(t, "0x574c65fda9065288556bde1eccf40afd32244330", staticExtra.ALM)

	tracker := NewPoolTracker(cfg, client)
	p, err := tracker.GetNewPoolState(context.Background(), pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotEqual(t, "{}", p.Extra)
	require.NotZero(t, p.BlockNumber)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.True(t, extra.Collateral.Sign() > 0)
	require.True(t, extra.PriceWad.Sign() > 0)

	p.Tokens[0].Decimals = 18
	p.Tokens[1].Decimals = 8
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	kai, wbtc := p.Tokens[0].Address, p.Tokens[1].Address

	amountKAI, _ := new(big.Int).SetString("100000000000000000000", 10) // 100 KAI net
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: kai, Amount: amountKAI},
		TokenOut:      wbtc,
	})
	require.NoError(t, err)
	require.True(t, res.TokenAmountOut.Amount.Sign() > 0)
	t.Logf("block=%d deleverage 100 KAI -> %s WBTC sat", p.BlockNumber, res.TokenAmountOut.Amount)

	res, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wbtc, Amount: big.NewInt(500_000)}, // 0.005 WBTC
		TokenOut:      kai,
	})
	require.NoError(t, err)
	require.True(t, res.TokenAmountOut.Amount.Sign() > 0)
	t.Logf("block=%d leverage 0.005 WBTC -> %s KAI wei", p.BlockNumber, res.TokenAmountOut.Amount)
}
