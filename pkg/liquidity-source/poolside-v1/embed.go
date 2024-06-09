package poolsidev1

import _ "embed"

//go:embed abis/ButtonswapPair.json
var pairABIJson []byte

//go:embed abis/ButtonswapFactory.json
var factoryABIJson []byte

//go:embed abis/ButtonToken.json
var buttonTokenABIJson []byte
