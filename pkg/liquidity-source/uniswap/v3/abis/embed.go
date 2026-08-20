package abis

import (
	_ "embed"
)

//go:embed UniswapV3Pool.json
var uniswapV3PoolJson []byte

//go:embed Slot0Slipstream.json
var slot0SlipstreamJson []byte

//go:embed Slot0Solidly.json
var slot0SolidlyJson []byte

//go:embed Slot0Katana.json
var slot0KatanaJson []byte

//go:embed UniswapV3Factory.json
var uniswapV3FactoryJson []byte
