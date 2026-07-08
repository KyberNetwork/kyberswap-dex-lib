package st0x

import _ "embed"

//go:embed abis/PropAMMHook.json
var propAMMHookABIJson []byte

//go:embed abis/PriceOracle.json
var priceOracleABIJson []byte
