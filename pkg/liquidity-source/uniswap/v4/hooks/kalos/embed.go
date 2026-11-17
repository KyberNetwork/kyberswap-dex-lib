package kalos

import _ "embed"

//go:embed abis/KalosDynamic.json
var kalosDynamicABIJson []byte

//go:embed abis/PoolManager.json
var poolManagerABIJson []byte
