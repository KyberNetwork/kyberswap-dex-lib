package few_v1

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// fewTokens lists Ring Protocol FewToken wrapper pools that predate the current
// Ring manifest generation (see few/v2 for the actively-published pool set).
// Kept as its own generation because at least one pair (WBTC/fwWBTC) has a
// distinct, older pool address/hook that is not superseded/confirmed dead, and
// packages must not silently shadow one generation's pool with another's.
var fewTokens = []few.TokenInfo{
	{
		// WBTC/fwWBTC (v1 pool, tickSpacing=60): https://etherscan.io/address/0x884c00abc9b0fa843ea2dfdd025e1df5611db552f396e2f17c88fb2ceef199e1
		ChainID:            valueobject.ChainIDEthereum,
		PoolAddress:        "0x884c00abc9b0fa843ea2dfdd025e1df5611db552f396e2f17c88fb2ceef199e1",
		HookAddress:        "0x948922b055187c7366e71b876ab1242ebbaea888",
		UnwrapTokenAddress: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		FewTokenAddress:    "0x2078f336fdd260f708bec4a20c82b063274e1b23", // fwWBTC
		Fee:                0,
		TickSpacing:        60,
	},
}

func NewTokenWrapper() few.TokenWrapper {
	return few.NewTokenWrapper(fewTokens)
}

// HookAddresses returns every hook address these pools use, for registration
// by the few/hook package (which can't live here: uniswapv4 already imports
// this package for NewTokenWrapper above, so this package can't import
// uniswapv4 back without an import cycle).
func HookAddresses() []string {
	addresses := make([]string, len(fewTokens))
	for i, t := range fewTokens {
		addresses[i] = t.HookAddress
	}
	return addresses
}
