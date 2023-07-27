package algebrav1

import (
	"fmt"
	"math/big"
	"strconv"
)

type Gas struct {
	Swap int64
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

type GlobalState struct {
	Price              *big.Int `json:"price"`
	Tick               *big.Int `json:"tick"`
	Fee                uint16   `json:"fee"`
	TimepointIndex     uint16   `json:"timepoint_index"`
	CommunityFeeToken0 uint8    `json:"community_fee_token0"`
	CommunityFeeToken1 uint8    `json:"community_fee_token1"`
	Unlocked           bool     `json:"unlocked"`
}

type FetchRPCResult struct {
	liquidity *big.Int
	state     GlobalState
	reserve0  *big.Int
	reserve1  *big.Int
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Extra struct {
	Liquidity                 *big.Int    `json:"liquidity"`
	VolumePerLiquidityInBlock *big.Int    `json:"volumePerLiquidityInBlock"`
	TotalFeeGrowth0Token      *big.Int    `json:"totalFeeGrowth0Token"`
	TotalFeeGrowth1Token      *big.Int    `json:"totalFeeGrowth1Token"`
	GlobalState               GlobalState `json:"globalState"`
	Ticks                     []Tick      `json:"ticks"`
}

type populatedTick struct {
	Tick           *big.Int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
}

func transformTickRespToTick(tickResp TickResp) (Tick, error) {
	liquidityGross := new(big.Int)
	liquidityGross, ok := liquidityGross.SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet := new(big.Int)
	liquidityNet, ok = liquidityNet.SetString(tickResp.LiquidityNet, 10)
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
