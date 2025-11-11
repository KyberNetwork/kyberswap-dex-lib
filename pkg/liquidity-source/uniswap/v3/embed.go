package uniswapv3

import (
	_ "embed"
)

//go:embed pregenesispools/optimism.json
var optimismPreGenesisPoolsBytes []byte

var BytesByPath = map[string][]byte{
	"pregenesispools/optimism.json": optimismPreGenesisPoolsBytes,
}
