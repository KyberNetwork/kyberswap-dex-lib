package nightlyprop

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	arbitrumRouter       = "0xeDE7B21238AB8D2bc7c5E62D7e6D141cAba5cbA2"
	arbitrumWethUsdcPool = "0x4dcfdd8afe7cde1fd591d6ac37c51f32277bdf5e"
	arbitrumWeth         = "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"
	arbitrumUsdc         = "0xaf88d065e77c8cc2239327c5edb3a432268e5831"
)

func TestNewPoolsListUpdater_InvalidRouterAddress(t *testing.T) {
	for _, address := range []string{"", "0x1234", valueobject.ZeroAddress, valueobject.ZeroAddress[2:]} {
		_, err := NewPoolsListUpdater(&Config{DexID: DexType, RouterAddress: address}, nil)
		require.ErrorIs(t, err, ErrInvalidRouterAddress, "router %q", address)
	}
}

func TestLivePoolsListUpdater(t *testing.T) {
	client := testutil.DialLive(t, testutil.LiveURL(t, "NIGHTLY_PROP_ARBITRUM_RPC_URL"))
	u, err := NewPoolsListUpdater(&Config{
		DexID:         DexType,
		ChainID:       valueobject.ChainIDArbitrumOne,
		RouterAddress: arbitrumRouter,
	}, client.Ethrpc)
	require.NoError(t, err)

	pools, cursor, err := u.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	for _, p := range pools {
		t.Logf("pool=%s base=%s quote=%s", p.Address, p.Tokens[0].Address, p.Tokens[1].Address)
		require.Equal(t, DexType, p.Type)
		require.JSONEq(t, `{"routerAddress":"0xede7b21238ab8d2bc7c5e62d7e6d141caba5cba2"}`, p.StaticExtra)
	}
	wethUsdc, ok := lo.Find(pools, func(p entity.Pool) bool { return p.Address == arbitrumWethUsdcPool })
	require.True(t, ok, "router does not list the WETH/USDC pool")
	require.Equal(t, arbitrumWeth, wethUsdc.Tokens[0].Address)
	require.Equal(t, arbitrumUsdc, wethUsdc.Tokens[1].Address)

	again, _, err := u.GetNewPools(t.Context(), cursor)
	require.NoError(t, err)
	require.Empty(t, again)
}
