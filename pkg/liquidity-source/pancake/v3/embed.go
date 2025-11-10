package pancakev3

import (
	_ "embed"
)

//go:embed abis/PancakeV3Pool.json
var pancakeV3PoolJson []byte

//go:embed abis/PancakeV3Factory.json
var pancakeV3FactoryJson []byte
