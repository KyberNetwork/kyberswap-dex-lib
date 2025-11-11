package solidlyv3

import (
	_ "embed"
)

//go:embed abis/SolidlyV3Pool.json
var solidlyV3PoolJson []byte

//go:embed abis/SolidlyV3Factory.json
var solidlyV3FactoryJson []byte
