package mooniswap

import _ "embed"

//go:embed abi/Pool.json
var poolABIJson []byte

//go:embed abi/Factory.json
var factoryABIJson []byte
