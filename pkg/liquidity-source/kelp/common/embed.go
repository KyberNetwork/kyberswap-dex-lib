package common

import _ "embed"

//go:embed abis/LRTConfig.json
var lrtConfigABIJson []byte

//go:embed abis/LRTDepositPool.json
var lrtDepositPoolABIJson []byte

//go:embed abis/LRTOracle.json
var lrtOracleABIJson []byte

//go:embed abis/ERC20.json
var erc20ABIJson []byte
