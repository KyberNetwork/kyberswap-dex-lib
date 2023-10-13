package zkswapfinance

import _ "embed"

//go:embed abis/ZFPair.json
var pairABIJson []byte

//go:embed abis/ZFFactory.json
var factoryABIJson []byte
