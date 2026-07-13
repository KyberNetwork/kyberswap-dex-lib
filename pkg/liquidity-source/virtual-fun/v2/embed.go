package v2

import _ "embed"

//go:embed abi/Bonding.json
var bondingABIJson []byte

//go:embed abi/FactoryV2.json
var factoryABIJson []byte

//go:embed abi/PairV2.json
var pairABIJson []byte

//go:embed abi/RouterV2.json
var routerABIJson []byte
