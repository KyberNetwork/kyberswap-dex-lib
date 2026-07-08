package nadswap

import _ "embed"

//go:embed abis/NadFunFactory.json
var factoryJson []byte

//go:embed abis/NadFunPair.json
var pairJson []byte

//go:embed abis/FeeCollector.json
var feeCollectorJson []byte
