package balancerweighted

import "math/big"

var (
	DefaultGas   = Gas{Swap: 80000}
	BoneFloat, _ = new(big.Float).SetString("1000000000000000000")
	MaxInRatio   = big.NewInt(30) // 30% = 0.3
	MaxOutRatio  = big.NewInt(30) // 30% = 0.3
)
