package wildcard

import _ "embed"

//go:embed abis/Factory.json
var factoryABIData []byte

//go:embed abis/Pair.json
var pairABIData []byte

//go:embed abis/ERC20.json
var erc20ABIData []byte

//go:embed abis/Multicall.json
var multicallABIData []byte
