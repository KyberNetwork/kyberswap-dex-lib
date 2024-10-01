package nuriv2

import (
	_ "embed"
)

//go:embed abis/NuriV2Pool.json
var nuriV2PoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
