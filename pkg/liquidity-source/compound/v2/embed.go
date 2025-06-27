package v2

import _ "embed"

//go:embed abis/CErc20.json
var cTokenJson []byte

//go:embed abis/Comptroller.json
var comptrollerJson []byte
