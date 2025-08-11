package unibtc

import _ "embed"

//go:embed abis/VaultUniBTC.json
var vaultUniBTCABIJson []byte

//go:embed abis/VaultBrBTC.json
var vaultBrBTCABIJson []byte
