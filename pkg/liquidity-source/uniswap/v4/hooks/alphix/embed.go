package alphix

import _ "embed"

//go:embed abis/Hook.json
var alphixHookABIJson []byte

//go:embed abis/LvrFeeHook.json
var lvrFeeHookABIJson []byte

//go:embed abis/ProHook.json
var proHookABIJson []byte
