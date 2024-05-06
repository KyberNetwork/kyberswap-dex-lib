//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple Gas
//msgp:ignore KSElasticSwapInfo Metadata Token SubgraphPool TickResp SubgraphPoolTicks StaticExtra Tick Extra PoolState LiquidityState FetchRPCResult

package elastic

import (
	"fmt"
	"math/big"
	"strconv"
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

type TickResp struct {
	TickIdx        string `json:"tickIdx"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type SubgraphPoolTicks struct {
	ID    string     `json:"id"`
	Ticks []TickResp `json:"ticks"`
}

type StaticExtra struct {
	PoolId string `json:"poolId"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Extra struct {
	Liquidity     *big.Int `json:"liquidity"`
	ReinvestL     *big.Int `json:"reinvestL"`
	ReinvestLLast *big.Int `json:"reinvestLLast"`
	SqrtPriceX96  *big.Int `json:"sqrtPriceX96"`
	Tick          *big.Int `json:"tick"`
	Ticks         []Tick   `json:"ticks"`
}

type PoolState struct {
	SqrtP              *big.Int `json:"sqrtP"`
	CurrentTick        *big.Int `json:"currentTick"`
	NearestCurrentTick *big.Int `json:"nearestCurrentTick"`
	Locked             bool     `json:"locked"`
}

type LiquidityState struct {
	BaseL         *big.Int `json:"baseL"`
	ReinvestL     *big.Int `json:"reinvestL"`
	ReinvestLLast *big.Int `json:"reinvestLLast"`
}

type FetchRPCResult struct {
	liquidityState LiquidityState
	poolState      PoolState
	reserve0       *big.Int
	reserve1       *big.Int
}

func transformTickRespToTick(tickResp TickResp) (Tick, error) {
	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}
