package angletransmuter

import (
	_ "embed"
)

//go:embed abis/transmuter.json
var TransmuterJson []byte

//go:embed abis/pyth.json
var PythJson []byte
