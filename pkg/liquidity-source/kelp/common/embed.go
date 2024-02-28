package common

import _ "embed"

//go:embed abis/LRTConfig.json
var lrtConfigABIJson []byte

//go:embed abis/LRTDepositPool.json
var lrtDepositPoolABIJson []byte
