package zkswap

import _ "embed"

//go:embed abi/Pair.json
var pairABIJson []byte

//go:embed abi/Factory.json
var factoryABIJson []byte
