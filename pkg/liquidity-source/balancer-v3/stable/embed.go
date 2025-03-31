package stable

import _ "embed"

//go:embed abis/StablePool.json
var poolJson []byte

//go:embed abis/StableSurgeHook.json
var stableSurgeJson []byte
