package ramsesv2

import (
	_ "embed"
)

//go:embed abis/RamsesV2Pool.json
var ramsesV2PoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
