package usdfi

import _ "embed"

//go:embed abi/Pair.json
var pairABIData []byte

//go:embed abi/Factory.json
var factoryABIData []byte
