package pancakev3

import (
	_ "embed"
)

//go:embed abis/PancakeV3Pool.json
var pancakeV3PoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
