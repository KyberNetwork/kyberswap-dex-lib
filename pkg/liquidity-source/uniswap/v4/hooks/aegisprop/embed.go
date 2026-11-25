package aegisprop

import _ "embed"

//go:embed abis/AegisPropHook.json
var aegisPropHookABIJson []byte

//go:embed abis/AegisPropHookV1.json
var aegisPropHookV1ABIJson []byte
