package ambient

import _ "embed"

//go:embed abis/ERC20.json
var erc20ABIBytes []byte

//go:embed abis/query.json
var queryABIBytes []byte

//go:embed abis/multicall.json
var multicallABIBytes []byte
