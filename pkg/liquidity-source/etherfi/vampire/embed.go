package etherfivampire

import _ "embed"

//go:embed abi/CurvePlain.json
var curvePlainABIJson []byte

//go:embed abi/EETH.json
var eETHABIJson []byte

//go:embed abi/LiquidityPool.json
var liquidityPoolABIJson []byte

//go:embed abi/StETH.json
var stETHABIJson []byte

//go:embed abi/Vampire.json
var vampireABIJson []byte
