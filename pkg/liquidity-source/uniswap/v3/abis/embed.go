package abis

import (
	_ "embed"
)

//go:embed UniswapV3Pool.json
var uniswapV3PoolJson []byte

//go:embed UniswapV3Factory.json
var uniswapV3FactoryJson []byte
