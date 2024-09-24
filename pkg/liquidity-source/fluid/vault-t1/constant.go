package vaultT1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

const (
	DexType = "fluid-vault-t1"
)

var vaultLiquidationResolver = map[valueobject.ChainID]string{
	valueobject.ChainIDEthereum:    "0x6Cd1E75b524D3CCa4c3320436d6F09e24Dadd613",
	valueobject.ChainIDArbitrumOne: "0x27F0Cb52138e97A66295Eeed523c9698E1125Fa9",
	valueobject.ChainIDBase:        "0x0e85C7d3764343A2924D9cDC6Fea1695De3243cC",
}

const (
	// VaultLiquidationResolver methods
	VLRMethodGetAllSwapPaths    = "getAllSwapPaths"
	VLRMethodGetSwapForProtocol = "getSwapForProtocol"

	// ERC20 Token methods
	TokenMethodDecimals = "decimals"
	TokenMethodSymbol   = "symbol"
	TokenMethodName     = "name"
)

const (
	String1e27 = "1000000000000000000000000000"
)
