package dexLite

import _ "embed"

var (
	//go:embed abis/FluidDexLite.json
	fluidDexLiteABIBytes []byte

	//go:embed abis/CenterPrice.json
	centerPriceABIBytes []byte
)
