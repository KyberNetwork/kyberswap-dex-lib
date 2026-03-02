package launchpadv2

import _ "embed"

//go:embed abi/BondingV4.json
var bodingABIJson []byte

//go:embed abi/FactoryV2.json
var factoryABIJson []byte

//go:embed abi/PairV2.json
var pairABIJson []byte

//go:embed abi/RouterV2.json
var routerABIJson []byte
