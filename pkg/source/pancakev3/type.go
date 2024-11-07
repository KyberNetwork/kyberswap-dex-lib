package pancakev3

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ticklens"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
}

// SwapInfoBigInt present the after state of a swap
type SwapInfoBigInt struct {
	nextStateSqrtRatioX96 *big.Int
	nextStateLiquidity    *big.Int
	nextStateTickCurrent  int
}

type SwapInfo struct {
	nextStateSqrtRatioX96 *v3Utils.Uint160
	nextStateLiquidity    *v3Utils.Uint128
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

type TickResp = ticklens.TickResp

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

type TickU256 struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64   `json:"tickSpacing"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`
}

type ExtraTickU256 struct {
	Liquidity    *uint256.Int `json:"liquidity"`
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64       `json:"tickSpacing"`
	Tick         *int         `json:"tick"`
	Ticks        []TickU256   `json:"ticks"`
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
	Liquidity   *big.Int `json:"liquidity"`
	Slot0       Slot0    `json:"slot0"`
	TickSpacing *big.Int `json:"tickSpacing"`
	Reserve0    *big.Int `json:"reserve0"`
	Reserve1    *big.Int `json:"reserve1"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
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
