package integral

import (
	_ "embed"
)

//go:embed abis/Factory.json
var algebraFactoryJson []byte

//go:embed abis/PoolV10.json
var poolV10Json []byte

//go:embed abis/PoolV12.json
var poolV12Json []byte

//go:embed abis/BasePluginV2.json
var basePluginV2Json []byte

//go:embed abis/TickLens.json
var ticklenJson []byte
