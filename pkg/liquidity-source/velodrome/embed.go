package velodrome

import _ "embed"

//go:embed abis/Pair.json
var pairABIJson []byte

//go:embed abis/PairFactory.json
var pairFactoryABIJson []byte
