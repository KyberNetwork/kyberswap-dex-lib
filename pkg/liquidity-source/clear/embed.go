package clear

import _ "embed"

//go:embed abis/ClearSwap.json
var clearSwapABIJson []byte

//go:embed abis/ClearFactory.json
var clearFactoryABIJson []byte
