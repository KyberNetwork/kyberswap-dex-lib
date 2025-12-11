package valantisstex

import _ "embed"

//go:embed abis/SovereignPool.json
var sovereignPoolBytes []byte

//go:embed abis/SwapFeeModule.json
var swapFeeModuleBytes []byte
