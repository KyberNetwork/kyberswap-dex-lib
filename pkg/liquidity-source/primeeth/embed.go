package primeeth

import _ "embed"

//go:embed abis/LRTDepositPool.json
var lrtDepositPoolABIJson []byte

//go:embed abis/LRTConfig.json
var lrtConfigABIJson []byte

//go:embed abis/LRTOracle.json
var lrtOracleABIJson []byte
