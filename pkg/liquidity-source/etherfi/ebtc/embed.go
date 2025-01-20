package etherfiebtc

import _ "embed"

//go:embed abis/TellerWithMultiAssetSupport.json
var tellerABIData []byte

//go:embed abis/AccountantWithRateProviders.json
var accountantABIData []byte

//go:embed pools/ethereum.json
var ethereumPoolData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
