package promm

import (
	"math/big"
)

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type NextState struct {
	SqrtRatioX96 *big.Int
	Liquidity    *big.Int
	ReinvestL    *big.Int
	TickCurrent  int
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	ReinvestL    *big.Int `json:"reinvestL"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`
}
