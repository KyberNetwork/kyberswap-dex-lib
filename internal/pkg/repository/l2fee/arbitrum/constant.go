package arbitrum

import (
	"math/big"
)

const (
	arbGasInfoAddress          = "0x000000000000000000000000000000000000006c"
	methodGetL1BaseFeeEstimate = "getL1BaseFeeEstimate"
)

var (
	l1GasOverhead = big.NewInt(15600)
	l1GasPerPool  = big.NewInt(630)
)
