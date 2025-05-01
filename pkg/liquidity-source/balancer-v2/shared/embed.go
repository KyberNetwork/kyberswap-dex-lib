package shared

import _ "embed"

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/ProtocolFeesCollector.json
var protocolFeesCollectorJson []byte
