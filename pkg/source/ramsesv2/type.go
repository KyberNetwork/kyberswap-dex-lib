package ramsesv2

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
}

// RamsesV2SwapInfo present the after state of a swap
type RamsesV2SwapInfo struct {
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
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
	Token0             Token  `json:"token0"`
	Token1             Token  `json:"token1"`
}

type SubgraphPoolTicks struct {
	ID    string              `json:"id"`
	Ticks []ticklens.TickResp `json:"ticks"`
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
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	FeeTier      int64    `json:"feeTier"`
	TickSpacing  uint64   `json:"tickSpacing"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`
	Unlocked     bool     `json:"unlocked"`
}

type PoolMeta struct {
	PriceLimit  *big.Int `json:"priceLimit"`
	BlockNumber uint64   `json:"blockNumber"`
}

type Slot0 struct {
	SqrtPriceX96               *big.Int `json:"sqrtPriceX96"`
	Tick                       *big.Int `json:"tick"`
	ObservationIndex           uint16   `json:"observationIndex"`
	ObservationCardinality     uint16   `json:"observationCardinality"`
	ObservationCardinalityNext uint16   `json:"observationCardinalityNext"`
	FeeProtocol                uint32   `json:"feeProtocol"`
	Unlocked                   bool     `json:"unlocked"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int
	Slot0       Slot0
	FeeTier     int64
	TickSpacing uint64
	Reserve0    *big.Int
	Reserve1    *big.Int
	BlockNumber uint64
}

func transformTickRespToTick(tickResp ticklens.TickResp) (Tick, error) {
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
