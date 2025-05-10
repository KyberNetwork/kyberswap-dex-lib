package hyeth

import _ "embed"

//go:embed pools/ethereum.json
var ethereumPoolData []byte

//go:embed abis/hyeth.json
var hyethABIData []byte

//go:embed abis/pool.json
var poolABIData []byte

//go:embed abis/issuance_module.json
var issuanceModuleABIData []byte

//go:embed abis/hyeth_component_4626.json
var hyethComponent4626ABIData []byte

var bytesByPath = map[string][]byte{
	"pools/ethereum.json": ethereumPoolData,
}
