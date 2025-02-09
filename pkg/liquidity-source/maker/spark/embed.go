package spark

import _ "embed"

//go:embed abis/Pot.json
var potJSON []byte

//go:embed abis/Savings.json
var savingsJSON []byte
