package v1

import (
	"bytes"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TestGetPoolDataQuotesTheFixedAmount: every oracle quote getPoolData asks for, a pool's vault, the controller
// and a collateral outside the pool alike, is for shared.QuoteAmount, no call reads decimals(), and every quote
// is turned into a price per smallest unit by dividing by that amount. The account's controller 0xc1 and its
// collateral 0xc2 are not vaults of the pool; each oracle answers getQuotes(a, ...) with (2a, 3a).
func TestGetPoolDataQuotesTheFixedAmount(t *testing.T) {
	var (
		controller, collateral = common.HexToAddress("0xc1"), common.HexToAddress("0xc2")
		oracle, unit, asset    = common.HexToAddress("0xaa"), common.HexToAddress("0xbb"), common.HexToAddress("0xdd")
		word                   = func(n uint64) []byte { return common.LeftPadBytes(new(big.Int).SetUint64(n).Bytes(), 32) }
		addrs                  = func(a ...common.Address) []byte {
			out := append(word(32), word(uint64(len(a)))...)
			for _, x := range a {
				out = append(out, common.LeftPadBytes(x.Bytes(), 32)...)
			}
			return out
		}
	)
	sel := func(name string, abis ...string) []byte { // a vault's method unless the ABI is named
		if len(abis) > 0 && abis[0] == "evc" {
			return evcABI.Methods[name].ID
		}
		if len(abis) > 0 && abis[0] == "router" {
			return routerABI.Methods[name].ID
		}
		return vaultABI.Methods[name].ID
	}
	var (
		mu       sync.Mutex
		amounts  []*big.Int
		decimals int
	)
	stub := testutil.NewMulticallStub(t, func(_ common.Address, data []byte) (bool, []byte) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case bytes.HasPrefix(data, sel("getCollaterals", "evc")):
			return true, addrs(collateral)
		case bytes.HasPrefix(data, sel("getControllers", "evc")):
			return true, addrs(controller)
		case bytes.HasPrefix(data, sel("oracle")):
			return true, common.LeftPadBytes(oracle.Bytes(), 32)
		case bytes.HasPrefix(data, sel("unitOfAccount")):
			return true, common.LeftPadBytes(unit.Bytes(), 32)
		case bytes.HasPrefix(data, sel("asset")):
			return true, common.LeftPadBytes(asset.Bytes(), 32)
		case bytes.HasPrefix(data, sel("decimals")):
			decimals++
			return true, word(18)
		case bytes.HasPrefix(data, sel(shared.RouterMethodGetQuotes, "router")):
			in := new(big.Int).SetBytes(data[4:36])
			amounts = append(amounts, in)
			return true, append(common.LeftPadBytes(new(big.Int).Mul(in, big.NewInt(2)).Bytes(), 32),
				common.LeftPadBytes(new(big.Int).Mul(in, big.NewInt(3)).Bytes(), 32)...)
		}
		// Anything else (balances, caps, reserves, LTVs, flags) is a few words of 1.
		return true, append(append(word(1), word(1)...), word(1)...)
	})
	tracker, err := NewPoolTracker(&shared.Config{}, stub.Client())
	require.NoError(t, err)

	vaults := []shared.VaultInfo{
		{VaultAddress: "0x0000000000000000000000000000000000000001", AssetAddress: "0x00000000000000000000000000000000000000a1"},
		{VaultAddress: "0x0000000000000000000000000000000000000002", AssetAddress: "0x00000000000000000000000000000000000000a2"},
	}
	data, _, err := tracker.getPoolData(t.Context(), "0x00000000000000000000000000000000000000f0",
		"0x00000000000000000000000000000000000000e0", "0x00000000000000000000000000000000000000c0", vaults, nil)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Zero(t, decimals, "decimals() was read")
	// 3 vaults (the pool's 2 and the controller) with an oracle each, quoting 3 vaults and 1 collateral.
	require.Len(t, amounts, 3*(3+1))
	for i, a := range amounts {
		assert.Equal(t, shared.QuoteAmount.String(), a.String(), "quote %d", i)
	}
	// Quotes of (2a, 3a) are prices of 2e18 and 3e18 per smallest unit: the divisor is the amount quoted.
	for j := range 3 {
		for i := range 3 {
			assert.Equal(t, "2000000000000000000", data.VaultPrices[j][i][0].String(), "vault %d in %d", j, i)
			assert.Equal(t, "3000000000000000000", data.VaultPrices[j][i][1].String(), "vault %d in %d", j, i)
		}
	}
	for i := range 3 {
		assert.Equal(t, "2000000000000000000", data.CollatPrices[0][i][0].String(), "collateral in %d", i)
		assert.Equal(t, "3000000000000000000", data.CollatPrices[0][i][1].String(), "collateral in %d", i)
	}
}
