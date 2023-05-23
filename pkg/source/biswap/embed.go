package biswap

import _ "embed"

//go:embed abis/BiswapPair.json
var pairABIJson []byte

//go:embed abis/BiswapFactory.json
var factoryABIJson []byte
