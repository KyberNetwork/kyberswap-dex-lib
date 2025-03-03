package honey

import _ "embed"

//go:embed pools/berachain.json
var berachainPoolData []byte

//go:embed abis/asset_vault.json
var assetVaultABIData []byte

//go:embed abis/honey_factory.json
var honeyFactoryABIData []byte

//go:embed abis/ERC20.json
var erc20ABIData []byte

var bytesByPath = map[string][]byte{
	"pools/berachain.json": berachainPoolData,
}
