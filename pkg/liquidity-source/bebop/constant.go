package bebop

import (
	"math/big"
)

const DexType = "bebop"

var (
	zeroBF     = big.NewFloat(0)
	defaultGas = Gas{Quote: 300000}
)
