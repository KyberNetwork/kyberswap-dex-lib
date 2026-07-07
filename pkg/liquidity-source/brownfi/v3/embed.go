package brownfiv3

import _ "embed"

//go:embed abis/BrownFiV3Factory.json
var factoryABIJson []byte

//go:embed abis/BrownFiV3Pair.json
var pairABIJson []byte

//go:embed abis/BrownFiV3PairConfig.json
var pairConfigABIJson []byte

//go:embed abis/BrownFiV3Oracle.json
var oracleABIJson []byte
