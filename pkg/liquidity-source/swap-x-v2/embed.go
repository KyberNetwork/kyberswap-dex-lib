package swapxv2

import _ "embed"

//go:embed abis/Pool.json
var poolABIJson []byte

//go:embed abis/Factory.json
var factoryABIJson []byte
