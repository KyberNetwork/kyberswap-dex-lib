package elastic

import (
	"math/big"
)

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

// KSElasticSwapInfo store after state of a KSElastic swap
type KSElasticSwapInfo struct {
	nextStateSqrtP              *big.Int
	nextStateBaseL              *big.Int
	nextStateReinvestL          *big.Int
	nextStateCurrentTick        int
	nextStateNearestCurrentTick int
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
