package aegis

import _ "embed"

//go:embed abis/Hook.json
var aegisHookABIJson []byte

//go:embed abis/DynamicFeeManager.json
var aegisDynamicFeeManagerABIJson []byte

//go:embed abis/PoolPolicyManager.json
var aegisPoolPolicyManagerABIJson []byte
