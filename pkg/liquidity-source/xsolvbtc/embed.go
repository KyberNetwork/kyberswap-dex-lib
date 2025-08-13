package xsolvbtc

import _ "embed"

//go:embed abis/Pool.json
var poolABIJson []byte

//go:embed abis/XsolvBTC.json
var xsolvBTCABIJson []byte
