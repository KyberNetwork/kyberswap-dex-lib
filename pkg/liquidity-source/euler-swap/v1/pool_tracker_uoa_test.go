package v1

import (
	"context"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/internal/rpcreplay"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestPricesUseTheVaultsUnitOfAccount tracks the Ethereum WETH/EUL pool 0xce8d8fd2 at block 26375800 from a fixture
// of the node's answers. A vault values its collateral with its own oracle in its own unit of account: the WETH
// vault in USD (0x348), the EUL vault in EUL. The tracker asked each oracle for the other vault's unit, so the WETH
// vault's oracle quoted EUL in EUL, returned the amount itself (1e18), and the tracker stored it as EUL's value in
// USD. That oracle cannot price EUL in USD (PriceOracle_NotSupported), so the price must stay 0.
//
// Set EULER_RECORD_RPC to a node at that block to record missing calls into the fixture.
func TestPricesUseTheVaultsUnitOfAccount(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/ethereum-26375800-0xce8d8fd2.json", os.Getenv("EULER_RECORD_RPC"))
	defer srv.Save()
	ctx := context.Background()
	client := srv.Client()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, client).initPools(ctx,
		[]common.Address{common.HexToAddress("0xce8d8fd259ff096763194e27be0578669db928a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	wethVault, eulVault := extra.Vaults[0], extra.Vaults[1]
	// The WETH vault's oracle: getQuote(1e18, WETH, USD) is 2084.63e18; EUL in USD is not supported.
	require.Equal(t, "2084630000000000000000", wethVault.VaultValuePrices[0].Dec())
	require.True(t, wethVault.VaultValuePrices[1].IsZero(), "EUL in the WETH vault: %s", wethVault.VaultValuePrices[1])
	// The EUL vault's oracle: EUL in EUL is 1e18; WETH in EUL is not supported.
	require.True(t, eulVault.VaultValuePrices[0].IsZero(), "WETH in the EUL vault: %s", eulVault.VaultValuePrices[0])
	require.Equal(t, "1000000000000000000", eulVault.VaultValuePrices[1].Dec())
}
