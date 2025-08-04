package dexLite

import _ "embed"

var (
	//go:embed abis/FluidDexLite.json
	fluidDexLiteABIBytes []byte

	//go:embed abis/ERC20.json
	erc20ABIBytes []byte
)
