package tidefiprop

import _ "embed"

//go:embed abis/TideFiSwapper.json
var swapperABIData []byte

//go:embed abis/ERC20.json
var erc20ABIData []byte
