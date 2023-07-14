package usdfi

import _ "embed"

//go:embed abis/Pair.json
var pairABIData []byte

//go:embed abis/Factory.json
var factoryABIData []byte
