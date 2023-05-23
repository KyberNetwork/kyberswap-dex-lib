package dmm

import _ "embed"

//go:embed abis/DmmPool.json
var poolABIJson []byte

//go:embed abis/DmmFactory.json
var factoryABIJson []byte
