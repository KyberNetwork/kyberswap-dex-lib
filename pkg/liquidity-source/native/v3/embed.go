package v3

import (
	_ "embed"
)

//go:embed abis/NativeV3Pool.json
var poolJson []byte

//go:embed abis/NativeV3Factory.json
var factoryJson []byte

//go:embed abis/NativeLPToken.json
var lpTokenJson []byte
