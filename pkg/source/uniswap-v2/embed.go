package uniswapv2

import _ "embed"

//go:embed abis/UniswapV2Pair.json
var pairABIJson []byte

//go:embed abis/UniswapV2Factory.json
var factoryABIJson []byte
