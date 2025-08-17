package shared

import _ "embed"

//go:embed abis/VaultExplorer.json
var vaultExplorerJson []byte

//go:embed abis/ERC4626.json
var erc4626Json []byte
