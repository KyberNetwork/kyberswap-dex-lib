package staderethx

import _ "embed"

//go:embed abis/StaderStakePoolsManager.json
var staderStakePoolsManagerABIJson []byte

//go:embed abis/StaderOracle.json
var staderOracleABIJson []byte
