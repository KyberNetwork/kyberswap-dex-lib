package brownfiv2

import _ "embed"

//go:embed abis/BrownFiV2Factory.json
var factoryABIJson []byte

//go:embed abis/BrownFiV2Pair.json
var pairABIJson []byte

//go:embed abis/BrownFiV2Oracle.json
var oracleABIJson []byte
