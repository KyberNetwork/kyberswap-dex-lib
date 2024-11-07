package fulcrom

import _ "embed"

//go:embed abis/Vault.json
var vaultJson []byte

//go:embed abis/VaultPriceFeed.json
var vaultPriceFeedJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
