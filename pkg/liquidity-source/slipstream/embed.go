package slipstream

import (
	_ "embed"
)

//go:embed abis/Pool.json
var poolJson []byte

//go:embed abis/Factory.json
var factoryJson []byte
