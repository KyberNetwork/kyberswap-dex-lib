package abis

import (
	_ "embed"
)

//go:embed DexV2.json
var dexV2Json []byte

//go:embed Liquidity.json
var liquidityJson []byte

//go:embed Resolver.json
var resolverJson []byte
