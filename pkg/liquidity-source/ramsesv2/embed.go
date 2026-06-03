package ramsesv2

import (
	_ "embed"
)

//go:embed abis/PoolV2.json
var ramsesV2PoolJson []byte

//go:embed abis/PoolV3.json
var ramsesV3PoolJson []byte

//go:embed abis/FactoryV2.json
var factoryV2Json []byte
