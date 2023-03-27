package balancerstable

import "math/big"

var (
	DefaultGas   = Gas{Swap: 80000}
	BoneFloat, _ = new(big.Float).SetString("1000000000000000000")
)
