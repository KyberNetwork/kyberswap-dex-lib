package integral

import (
	_ "embed"
)

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed abis/AlgebraPool.json
var algebraIntegralPoolJson []byte

//go:embed abis/AlgebraBasePluginV1.json
var algebraBasePluginV1Json []byte

//go:embed abis/TickLens.json
var ticklenJson []byte
