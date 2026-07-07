package dexT1

import _ "embed"

//go:embed abis/dexReservesResolver.json
var dexReservesResolverJSON []byte

//go:embed abis/ERC20.json
var erc20JSON []byte

//go:embed abis/storageRead.json
var storageReadJSON []byte
