package hashflowv3

import "math/big"

const DexType = "hashflow-v3"

var (
	zeroBF            = big.NewFloat(0)
	defaultGas        = Gas{Quote: 300000}
	priceToleranceBps = 10000
)
