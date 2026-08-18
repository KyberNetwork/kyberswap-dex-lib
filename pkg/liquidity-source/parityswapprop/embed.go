package parityswapprop

import _ "embed"

//go:embed abi/PmmRegistry.json
var pmmRegistryBytes []byte

//go:embed abi/PmmPool.json
var pmmPoolBytes []byte

//go:embed abi/PmmOracle.json
var pmmOracleBytes []byte
