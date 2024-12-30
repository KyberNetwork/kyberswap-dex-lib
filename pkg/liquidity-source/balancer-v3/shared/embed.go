package shared

import _ "embed"

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/VaultExtension.json
var vaultExtensionJson []byte
