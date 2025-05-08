package alphafee

import "math/big"

type pathReduction struct {
	PathIdx      int
	ReduceAmount *big.Int
}

type swapInfoV2 struct {
	Pool      string
	TokenIn   string
	TokenOut  string
	AmountIn  *big.Int
	AmountOut *big.Int
	Exchange  string
}
