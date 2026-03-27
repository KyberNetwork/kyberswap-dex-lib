package poe

import _ "embed"

//go:embed abi/Pool.json
var poolABIJson []byte

//go:embed abi/Oracle.json
var oracleABIJson []byte
