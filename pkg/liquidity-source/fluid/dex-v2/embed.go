package dexv2

import (
	_ "embed"
)

//go:embed abis/Liquidity.json
var liquidityJson []byte

//go:embed abis/Resolver.json
var resolverJson []byte
