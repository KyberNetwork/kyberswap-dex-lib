package fourmeme

import _ "embed"

//go:embed abis/TokenManager.json
var tokenManagerABIJson []byte

//go:embed abis/TokenManager2.json
var tokenManager2ABIJson []byte

//go:embed abis/TokenManagerHelper3.json
var tokenManagerHelperABIJson []byte
