package pufeth

import _ "embed"

//go:embed abis/PufferVault.json
var pufferVaultABIJson []byte

//go:embed abis/Lido.json
var lidoABIJson []byte
