package uniswapv2

import _ "embed"

//go:embed abis/UniswapV2Pair.json
var pairABIJson []byte

//go:embed abis/UniswapV2Factory.json
var factoryABIJson []byte

//go:embed abis/MeerkatPair.json
var meerkatPairABIJson []byte

//go:embed abis/MdexFactory.json
var mdexFactoryABIJson []byte

//go:embed abis/ShibaswapPair.json
var shibaswapPairABIJson []byte

//go:embed abis/CroDefiSwapFactory.json
var croDefiSwapFactoryABIJson []byte
