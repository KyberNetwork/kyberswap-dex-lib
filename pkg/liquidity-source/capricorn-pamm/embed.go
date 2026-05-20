package capricornpamm

import _ "embed"

//go:embed abi/PAMMPool.json
var pammPoolBytes []byte

//go:embed abi/PricingEngine.json
var pricingEngineBytes []byte

//go:embed abi/OracleRegistry.json
var oracleRegistryBytes []byte
