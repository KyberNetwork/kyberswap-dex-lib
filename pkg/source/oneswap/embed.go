package oneswap

import _ "embed"

//go:embed abi/OneSwapFactory.json
var oneSwapFactoryABIData []byte

//go:embed abi/OneSwap.json
var oneSwapABIData []byte

//go:embed abi/ERC20.json
var erc20Json []byte
