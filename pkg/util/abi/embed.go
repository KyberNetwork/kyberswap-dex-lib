package abi

import (
	_ "embed"
)

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed abis/Multicall3.json
var multicall3Json []byte
