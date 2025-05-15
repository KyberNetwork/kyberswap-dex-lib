package brownfi

import _ "embed"

//go:embed abis/BrownFiV1Pair.json
var pairABIJson []byte

//go:embed abis/BrownFiV1Factory.json
var factoryABIJson []byte
