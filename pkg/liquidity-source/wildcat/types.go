package wildcat

import "math/big"

type Extra struct {
	Rates    []*big.Int      `json:"rates"`
	IsNative []bool          `json:"isNative"`
	Samples  [][][2]*big.Int `json:"samples"` // [2]*big.Int = [amountIn, amountOut]
}

type PoolExtra struct {
	TokenInIsNative  bool `json:"tokenInIsNative"`
	TokenOutIsNative bool `json:"tokenOutIsNative"`
}
