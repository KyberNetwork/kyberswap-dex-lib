package traderjoev20

import _ "embed"

//go:embed abis/LBPair.json
var pairABIJson []byte

//go:embed abis/LBFactory.json
var factoryABIJson []byte

//go:embed abis/LBRouter.json
var routerABIJson []byte
