package rsethl2

import _ "embed"

//go:embed abis/LRTDepositPool.json
var LRTDepositPoolABIData []byte

//go:embed abis/LRTOracle.json
var LRTOracleABIData []byte
