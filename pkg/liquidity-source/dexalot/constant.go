package dexalot

import (
	"math/big"
)

const DexType = "dexalot"

var (
	zeroBF     = big.NewFloat(0)
	defaultGas = Gas{Quote: 200000}
)
