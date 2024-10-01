package slipstream

import (
	_ "embed"
)

//go:embed abis/Pool.json
var poolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
