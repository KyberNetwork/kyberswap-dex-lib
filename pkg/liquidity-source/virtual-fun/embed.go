package virtualfun

import _ "embed"

//go:embed abis/bonding.json
var bodingABIJson []byte

//go:embed abis/FFactory.json
var factoryABIJson []byte

//go:embed abis/FPair.json
var pairABIJson []byte

//go:embed abis/FERC20.json
var erc20ABIJson []byte
