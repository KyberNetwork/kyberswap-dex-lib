package uniswapv4

import "math/big"

const DexType = "uniswap-v4"
const EMPTY_ADDRESS = "0x0000000000000000000000000000000000000000"

const (
	graphSkipLimit  = 5000
	graphFirstLimit = 1000
)

var Q96 = big.NewInt(1).Lsh(big.NewInt(1), 96)
