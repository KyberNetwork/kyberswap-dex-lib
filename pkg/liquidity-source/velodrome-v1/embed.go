package velodromev1

import _ "embed"

//go:embed abis/Pair.json
var pairABIJson []byte

//go:embed abis/PairFactory.json
var pairFactoryABIJson []byte

//go:embed abis/StratumPairFactory.json
var stratumPairFactoryABIJson []byte
