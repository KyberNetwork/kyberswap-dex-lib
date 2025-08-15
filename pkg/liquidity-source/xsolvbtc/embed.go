package xsolvbtc

import _ "embed"

//go:embed abis/Pool.json
var poolABIJson []byte

//go:embed abis/xsolvBTC.json
var xsolvBTCABIJson []byte
