package ambient

import _ "embed"

//go:embed abis/ERC20.json
var erc20ABIBytes []byte

//go:embed abis/multicall.json
var multicallABIBytes []byte
