package warpx

import _ "embed"

//go:embed abis/WarpV2Pair.json
var pairABIJson []byte

//go:embed abis/WarpFactory.json
var factoryABIJson []byte
