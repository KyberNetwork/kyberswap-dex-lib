package eethorweeth

import _ "embed"

//go:embed abi/eETH.json
var eETHABIJson []byte

//go:embed abi/liquidityPool.json
var liquidityPoolABIJson []byte

//go:embed abi/stETH.json
var stETHABIJson []byte

//go:embed abi/vampire.json
var vampireABIJson []byte
