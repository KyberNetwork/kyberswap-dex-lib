package etherfiebtc

import _ "embed"

//go:embed abis/TellerWithMultiAssetSupport.json
var tellerABIData []byte

//go:embed abis/AccountantWithRateProviders.json
var accountantABIData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed pools/arbitrum.json
var arbitrumPoolData []byte

//go:embed pools/base.json
var basePoolData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
	"pools/arbitrum.json": arbitrumPoolData,
	"pools/base.json":     basePoolData,
}
