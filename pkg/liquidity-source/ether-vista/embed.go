package ethervista

import _ "embed"

//go:embed abis/Factory.json
var factoryABIJson []byte

//go:embed abis/Pair.json
var pairABIJson []byte

//go:embed abis/Router.json
var routerABIJson []byte
