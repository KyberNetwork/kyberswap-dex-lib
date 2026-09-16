package stablestable

import _ "embed"

//go:embed abis/StableStableHook.json
var stableStableHookABIJson []byte

//go:embed abis/StableStableHookV2.json
var stableStableHookV2ABIJson []byte
