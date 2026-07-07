package wasabiprop

import _ "embed"

//go:embed abis/Factory.json
var factoryABIData []byte

//go:embed abis/Pool.json
var poolABIData []byte
