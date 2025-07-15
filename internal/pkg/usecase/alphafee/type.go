package alphafee

import "math/big"

type pathReduction struct {
	PathIdx      int
	ReduceAmount *big.Int
}

type pathExchangeRate struct {
	PathIdx        int
	PathAmountInF  float64
	PathAmountOutF float64
}

type swapInfoV2 struct {
	Pool      string
	TokenIn   string
	TokenOut  string
	AmountIn  *big.Int
	AmountOut *big.Int
	Exchange  string
}

type amountThroughPool struct {
	TotalAmountIn  *big.Int
	TotalAmountOut *big.Int
	Count          int
}
