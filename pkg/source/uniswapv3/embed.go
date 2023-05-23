package uniswapv3

import (
	_ "embed"
)

//go:embed abis/UniswapV3Pool.json
var uniswapV3PoolJson []byte

//go:embed abis/TickLensProxy.json
var tickLensProxyJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed pregenesispools/optimism.json
var optimismPreGenesisPoolsBytes []byte

var BytesByPath = map[string][]byte{
	"pregenesispools/optimism.json": optimismPreGenesisPoolsBytes,
}
