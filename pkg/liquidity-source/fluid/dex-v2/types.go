package dexv2

import (
	"math/big"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
}

type SwapInfo struct {
	RemainingAmountIn     *v3Utils.Int256  `json:"rAI,omitempty"`
	NextStateSqrtRatioX96 *v3Utils.Uint160 `json:"nSqrtRx96"`
	NextStateTickCurrent  int              `json:"nT"`
	nextStateLiquidity    *v3Utils.Uint128
	amountInRawAdjusted   *big.Int
	amountOutRawAdjusted  *big.Int
}

type PoolMeta struct {
	Dex         string `json:"dex"`
	ZeroForOne  bool   `json:"zeroForOne,omitempty"`
	DexType     uint32 `json:"dexType"`
	Fee         uint32 `json:"fee"`
	TickSpacing uint32 `json:"tickSpacing"`
	Controller  string `json:"controller,omitempty"`

	IsNativeIn  bool `json:"isNativeIn,omitempty"`
	IsNativeOut bool `json:"isNativeOut,omitempty"`
}

type Metadata struct {
	LastCreatedAtTimestamp int      `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"`
}

type SubgraphPool struct {
	ID          string `json:"id"`
	DexId       string `json:"dexId"`
	DexType     uint32 `json:"dexType"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Fee         uint32 `json:"fee"`
	TickSpacing uint32 `json:"tickSpacing"`
	Controller  string `json:"controller"`
	CreatedAt   int    `json:"createdAt"`
}

type TickResp struct {
	Tick           int    `json:"tick"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type StaticExtra struct {
	Dex         string  `json:"dex"`
	DexType     uint32  `json:"dexType"`
	Fee         uint32  `json:"fee"`
	TickSpacing uint32  `json:"tickSpacing"`
	Controller  string  `json:"controller,omitempty"`
	IsNative    [2]bool `json:"isNative"`
}

type Extra struct {
	Liquidity    *big.Int `json:"liquidity"`
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Tick         *big.Int `json:"tick"`
	Ticks        []Tick   `json:"ticks"`

	DexVariables2                 *big.Int `json:"dexVariables2"`
	Token0ExchangePricesAndConfig *big.Int `json:"token0ExchangePricesAndConfig"`
	Token1ExchangePricesAndConfig *big.Int `json:"token1ExchangePricesAndConfig"`

	TokenReserves *big.Int `json:"tokenReserves"`
}

type DexKey struct {
	Token0      common.Address
	Token1      common.Address
	Fee         *big.Int
	TickSpacing *big.Int
	Controller  common.Address
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
	PoolAccountingFlag             bool     `json:"poolAccountingFlag"`
	FetchDynamicFeeFlag            bool     `json:"fetchDynamicFeeFlag"`
	FeeVersion                     *big.Int `json:"feeVersion"`
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
	DexId           common.Hash     `json:"dexId"`
	DexPriceParsed  *big.Int        `json:"dexPriceParsed"`
	DexPoolStateRaw DexPoolStateRaw `json:"dexPoolStateRaw"`
}

type DexPoolStateRaw struct {
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

type CalculatedVars struct {
	Token0NumeratorPrecision   *big.Int
	Token0DenominatorPrecision *big.Int
	Token1NumeratorPrecision   *big.Int
	Token1DenominatorPrecision *big.Int

	Token0SupplyExchangePrice *big.Int
	Token0BorrowExchangePrice *big.Int
	Token1SupplyExchangePrice *big.Int
	Token1BorrowExchangePrice *big.Int
}

type DynamicFeeVariables struct {
	minFee                         *big.Int
	maxFee                         *big.Int
	priceImpactToFeeDivisionFactor *big.Int
	zeroPriceImpactPriceX96        *big.Int
	minFeeKinkPriceX96             *big.Int
	minFeeKinkSqrtPriceX96         *big.Int
	maxFeeKinkPriceX96             *big.Int
	maxFeeKinkSqrtPriceX96         *big.Int
}

type DynamicFeeVariablesUI struct {
	minFee                         *v3Utils.Uint256
	maxFee                         *v3Utils.Uint256
	priceImpactToFeeDivisionFactor *v3Utils.Uint256
	zeroPriceImpactPriceX96        *v3Utils.Uint256
	minFeeKinkPriceX96             *v3Utils.Uint256
	minFeeKinkSqrtPriceX96         *v3Utils.Uint256
	maxFeeKinkPriceX96             *v3Utils.Uint256
	maxFeeKinkSqrtPriceX96         *v3Utils.Uint256
}
