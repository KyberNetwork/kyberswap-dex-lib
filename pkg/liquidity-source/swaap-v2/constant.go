package swaapv2

import "math/big"

const (
	DexType = "swaap-v2"
)

var (
	DefaultGas = Gas{Swap: 100000}

	priceToleranceBps = big.NewInt(10000)
)
