package vaultT1

import _ "embed"

//go:embed abis/vaultLiquidationResolver.json
var vaultLiquidationResolverJSON []byte

//go:embed abis/ERC20.json
var erc20JSON []byte
