package aavev3

import _ "embed"

//go:embed abis/Pool.json
var poolJson []byte

//go:embed abis/AToken.json
var aTokenJson []byte
