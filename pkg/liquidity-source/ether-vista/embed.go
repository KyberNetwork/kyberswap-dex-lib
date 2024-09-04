package ethervista

import _ "embed"

//go:embed abis/Pair.json
var pairABIJson []byte

//go:embed abis/Factory.json
var factoryABIJson []byte
