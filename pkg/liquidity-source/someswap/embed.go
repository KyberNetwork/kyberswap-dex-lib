package someswap

import _ "embed"

//go:embed abis/SomePair.json
var pairABIJson []byte

//go:embed abis/SomeFactory.json
var factoryABIJson []byte
