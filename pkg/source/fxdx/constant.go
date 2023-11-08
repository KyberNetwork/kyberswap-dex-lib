package fxdx

import "math/big"

const (
	DexTypeFxdx = "fxdx"

	flagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"
)

var (
	DefaultGas         = Gas{Swap: 165000}
	BasisPointsDivisor = big.NewInt(10000)
	PricePrecision     = new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	OneUSD             = PricePrecision
	USDFDecimals       = big.NewInt(18)
)
