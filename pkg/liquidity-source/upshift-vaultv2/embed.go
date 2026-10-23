package upshiftvaultv2

import _ "embed"

//go:embed abi/Vault.json
var vaultABIJson []byte

//go:embed abi/AssetsWhitelist.json
var assetsWhitelistABIJson []byte

//go:embed abi/ERC20.json
var erc20ABIJson []byte
