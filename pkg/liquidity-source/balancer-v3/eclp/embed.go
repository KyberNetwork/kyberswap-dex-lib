package eclp

import _ "embed"

//go:embed abis/ECLPPool.json
var poolJson []byte

//go:embed abis/Vault.json
var vaultJson []byte
