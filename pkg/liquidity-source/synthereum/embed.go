package synthereum

import _ "embed"

//go:embed abis/SynthereumMultiLpLiquidityPool.json
var multiLpPoolABIData []byte

//go:embed abis/ERC4626Vault.json
var vaultABIData []byte

//go:embed pools/base.json
var basePoolData []byte

var bytesByPath = map[string][]byte{
	"pools/base.json": basePoolData,
}
