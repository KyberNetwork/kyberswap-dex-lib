package uniswapv1

import _ "embed"

//go:embed abis/uniswap_exchange.json
var exchangeABIJson []byte

//go:embed abis/uniswap_factory.json
var factoryABIJson []byte

//go:embed abis/multicall.json
var multicallABIJson []byte

//go:embed abis/ERC20.json
var erc20ABIJson []byte
