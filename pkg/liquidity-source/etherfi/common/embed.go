package common

import _ "embed"

//go:embed abis/eETH.json
var eETHABIJson []byte

//go:embed abis/LiquidityPool.json
var liquidityPoolABIJson []byte
