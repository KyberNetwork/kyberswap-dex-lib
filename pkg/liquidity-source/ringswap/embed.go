package ringswap

import _ "embed"

//go:embed abis/UniswapV2Pair.json
var pairABIJson []byte

//go:embed abis/UniswapV2Factory.json
var factoryABIJson []byte

//go:embed abis/IFewWrappedToken.json
var fewWrappedTokenABIJson []byte
