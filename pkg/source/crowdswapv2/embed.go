package crowdswapv2

import _ "embed"

//go:embed abis/CrowdswapV2Pair.json
var pairABIJson []byte

//go:embed abis/CrowdswapV2Factory.json
var factoryABIJson []byte
