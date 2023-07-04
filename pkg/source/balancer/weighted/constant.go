package balancerweighted

import "math/big"

var (
	MaxInRatio   = big.NewInt(30) // 30% = 0.3
	MaxOutRatio  = big.NewInt(30) // 30% = 0.3
)