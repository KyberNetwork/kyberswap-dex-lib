package v1

import (
	"context"
	"encoding/hex"
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
func TestPricesUseTheVaultsUnitOfAccount(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/ethereum-26375800-0xce8d8fd2.json")
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

// TestTrackerQuotesTheFixedAmount tracks the BSC USDT/USDC pool 0xcc1713f8 at block 131600018 (both tokens have 18
// decimals; the pool lister leaves the tokens' decimals unset) and checks what the tracker asked the node: every
// oracle quote is for shared.QuoteAmount, and no call reads decimals(), of a pool vault, the controller or a
// collateral. The prices are as when the quotes were for one whole unit: USDT at $0.99859525 in the 18-decimal USD
// unit of account is 998595250000000000 (per smallest unit, times 1e18), as the oracle's getQuotes(1e18, ...)
// of the fixture gives. Those getQuotes(1e18, ...) answers
// are recorded from a node.
func TestTrackerQuotesTheFixedAmount(t *testing.T) {
	srv := rpcreplay.New(t, "testdata/bsc-131600018-0xcc1713f8.json")
	ctx := context.Background()
	client := srv.Client()
	cfg := shared.Config{DexID: DexType}
	pools, err := NewPoolsListUpdater(&cfg, client).initPools(ctx,
		[]common.Address{common.HexToAddress("0xcc1713f8220fae2d63215a3f81ef95981423e8a8")})
	require.NoError(t, err)
	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(ctx, pools[0], poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)

	amounts := srv.QuotedAmounts()
	require.NotEmpty(t, amounts)
	for _, a := range amounts {
		require.Equal(t, shared.QuoteAmount.String(), a.String())
	}
	require.Zero(t, srv.Selectors()[hex.EncodeToString(vaultABI.Methods["decimals"].ID)], "decimals() was asked")

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.Equal(t, "998595250000000000", extra.Vaults[0].VaultValuePrices[0].Dec())
	require.Equal(t, "999750590000000000", extra.Vaults[0].VaultValuePrices[1].Dec())
}
