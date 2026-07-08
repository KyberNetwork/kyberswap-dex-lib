package unipool

import _ "embed"

//go:embed abis/UniPoolFactory.json
var factoryABIJson []byte

//go:embed abis/UniPoolPair.json
var pairABIJson []byte
