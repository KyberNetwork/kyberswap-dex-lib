package uniswapv3pt

import (
	"math/big"
)

type Gas struct {
	Swap int64
}

// UniV3SwapInfo present the after state of a swap
type UniV3SwapInfo struct {
	nextStateSqrtRatioX96 *big.Int
	nextStateLiquidity    *big.Int
	nextStateTickCurrent  int
}

type Metadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
}

type Token struct {
	Address  string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
	ID                 string `json:"id"`
	FeeTier            string `json:"feeTier"`
	PoolType           string `json:"poolType"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
	Token0             Token  `json:"token0"`
	Token1             Token  `json:"token1"`
}

type StaticExtra struct {
	PoolId string `json:"poolId"`
}

type TickResp struct {
	TickIdx        int      `json:"tickIdx"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`
}

type Slot0 struct {
	SqrtPriceX96               *big.Int `json:"sqrtPriceX96"`
	Tick                       *big.Int `json:"tick"`
	ObservationIndex           uint16   `json:"observationIndex"`
	ObservationCardinality     uint16   `json:"observationCardinality"`
	ObservationCardinalityNext uint16   `json:"observationCardinalityNext"`
	FeeProtocol                uint8    `json:"feeProtocol"`
	Unlocked                   bool     `json:"unlocked"`
}

type preGenesisPool struct {
	ID string `json:"id"`
}

type populatedTick struct {
	Tick           *big.Int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
}

type FetchRPCResult struct {
	liquidity *big.Int
	slot0     Slot0
	reserve0  *big.Int
	reserve1  *big.Int
}
