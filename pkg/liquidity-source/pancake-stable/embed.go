package pancakestable

import _ "embed"

//go:embed abis/PancakeStableSwapFactory.json
var factoryABIJson []byte

//go:embed abis/PancakeStableSwap.json
var poolABIJson []byte
