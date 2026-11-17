package btrprop

import _ "embed"

//go:embed abis/Pool.json
var poolABIJSONBytes []byte

//go:embed abis/YieldHook.json
var hookABIJSONBytes []byte
