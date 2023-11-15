package velodromev2

import _ "embed"

//go:embed abis/Pool.json
var poolABIJson []byte

//go:embed abis/PoolFactory.json
var poolFactoryABIJson []byte
