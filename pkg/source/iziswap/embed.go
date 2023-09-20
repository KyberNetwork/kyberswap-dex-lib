package iziswap

import (
	_ "embed"
)

//go:embed abis/iZiSwapPool.json
var iZiSwapPoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
