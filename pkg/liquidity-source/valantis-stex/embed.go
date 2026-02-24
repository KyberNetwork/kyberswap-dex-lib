package valantisstex

import _ "embed"

//go:embed abis/SovereignPool.json
var sovereignPoolBytes []byte

//go:embed abis/SwapFeeModule.json
var swapFeeModuleBytes []byte

//go:embed abis/StexAMM.json
var stexAMMBytes []byte

//go:embed abis/WithdrawalModule.json
var withdrawalModuleBytes []byte
