package dracula

import _ "embed"

//go:embed abi/DraculaPair.json
var pairABIData []byte

//go:embed abi/DraculaFactory.json
var factoryABIData []byte
