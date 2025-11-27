package dexv2

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
}

type SwapInfo struct {
	RemainingAmountIn     *v3Utils.Int256  `json:"rAI,omitempty"`
	NextStateSqrtRatioX96 *v3Utils.Uint160 `json:"nSqrtRx96"`
	nextStateLiquidity    *v3Utils.Uint128
	NextStateTickCurrent  int `json:"nT"`
}

type ExtraTickU256 struct {
	Liquidity    *uint256.Int `json:"liquidity"`
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	TickSpacing  uint64       `json:"tickSpacing"`
	Tick         *int         `json:"tick"`
	Ticks        []TickU256   `json:"ticks"`
}

type TickU256 struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

type PoolMeta struct {
	SwapFee    uint32       `json:"swapFee"`
	PriceLimit *uint256.Int `json:"priceLimit"`
}

type Metadata struct {
	LastCreatedAtTimestamp int      `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"`
}

type SubgraphPool struct {
	ID          string `json:"id"`
	DexId       string `json:"dexId"`
	DexType     int    `json:"dexType"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Fee         int    `json:"fee"`
	TickSpacing int    `json:"tickSpacing"`
	Controller  string `json:"controller"`
	CreatedAt   int    `json:"createdAt"`
}

type TickResp struct {
	Tick           int    `json:"tick"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type Extra struct {
	DexType     int     `json:"dexType"`
	Fee         int     `json:"fee"`
	TickSpacing int     `json:"tickSpacing"`
	Controller  string  `json:"controller,omitempty"`
	IsNative    [2]bool `json:"isNative"`
	Ticks       []Tick  `json:"ticks"`

	DexVariables  DexVariables  `json:"dexVariables"`
	DexVariables2 DexVariables2 `json:"dexVariables2"`
}

type DexVariables struct {
	CurrentTick          *big.Int `json:"currentTick"`
	CurrentSqrtPriceX96  *big.Int `json:"currentSqrtPriceX96"`
	FeeGrowthGlobal0X102 *big.Int `json:"feeGrowthGlobal0X102"`
	FeeGrowthGlobal1X102 *big.Int `json:"feeGrowthGlobal1X102"`
}

type DexVariables2 struct {
	ProtocolFee0To1                *big.Int `json:"protocolFee0To1"`
	ProtocolFee1To0                *big.Int `json:"protocolFee1To0"`
	ProtocolCutFee                 *big.Int `json:"protocolCutFee"`
	Token0Decimals                 *big.Int `json:"token0Decimals"`
	Token1Decimals                 *big.Int `json:"token1Decimals"`
	ActiveLiquidity                *big.Int `json:"activeLiquidity"`
	FetchDynamicFeeFlag            bool     `json:"fetchDynamicFeeFlag"`
	InbuiltDynamicFeeFlag          bool     `json:"inbuiltDynamicFeeFlag"`
	LpFee                          *big.Int `json:"lpFee"`
	MaxDecayTime                   *big.Int `json:"maxDecayTime"`
	PriceImpactToFeeDivisionFactor *big.Int `json:"priceImpactToFeeDivisionFactor"`
	MinFee                         *big.Int `json:"minFee"`
	MaxFee                         *big.Int `json:"maxFee"`
	NetPriceImpact                 *big.Int `json:"netPriceImpact"`
	LastUpdateTimestamp            *big.Int `json:"lastUpdateTimestamp"`
	DecayTimeRemaining             *big.Int `json:"decayTimeRemaining"`
}

type DexPoolState struct {
	DexVariablesPacked    *big.Int      `json:"dexVariablesPacked"`
	DexVariables2Packed   *big.Int      `json:"dexVariables2Packed"`
	DexVariablesUnpacked  DexVariables  `json:"dexVariablesUnpacked"`
	DexVariables2Unpacked DexVariables2 `json:"dexVariables2Unpacked"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}
