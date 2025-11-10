package ramsesv2

import (
	_ "embed"
)

//go:embed abis/RamsesV2Pool.json
var ramsesV2PoolJson []byte

//go:embed abis/RamsesV3Pool.json
var ramsesV3PoolJson []byte

//go:embed abis/FactoryV2.json
var factoryV2Json []byte

//go:embed  abis/FactoryV3.json
var factoryV3Json []byte

//go:embed  abis/PharaohV3Pool.json
var pharaohV3PoolJson []byte
