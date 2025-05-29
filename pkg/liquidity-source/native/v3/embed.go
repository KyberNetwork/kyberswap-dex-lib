package v3

import (
	_ "embed"
)

//go:embed abis/NativeV3Pool.json
var poolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed abis/NativeLPToken.json
var lpTokenJson []byte
