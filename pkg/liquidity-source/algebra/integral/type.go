package integral

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/holiman/uint256"
)

type Metadata struct {
	LastCreatedAtTimestamp *big.Int `json:"lastCreatedAtTimestamp"`
	LastPoolIds            []string `json:"lastPoolIds"` // pools that share lastCreatedAtTimestamp
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

type Tick struct {
	LiquidityTotal       *big.Int
	LiquidityDelta       *big.Int
	PrevTick             *big.Int
	NextTick             *big.Int
	OuterFeeGrowth0Token *big.Int
	OuterFeeGrowth1Token *big.Int
}

func (t TickResp) transformTickRespToTick() (v3Entities.Tick, error) {
	liquidityGross, err := uint256.FromDecimal(t.LiquidityGross)
	if err != nil {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityGross string to uint256, tick: %v", t.TickIdx)
	}

	liquidityNet, err := int256.FromDec(t.LiquidityNet)
	if err != nil {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityNet string to uint256, tick: %v", t.TickIdx)
	}

	tickIdx, err := strconv.Atoi(t.TickIdx)
	if err != nil {
		return v3Entities.Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", t.TickIdx)
	}

	return v3Entities.Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}

type SubgraphPoolTicks struct {
	ID    string     `json:"id"`
	Ticks []TickResp `json:"ticks"`
}

// GlobalStateFromRPC for algebra v1 with single fee for both direction
type GlobalStateFromRPC struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}

// GlobalState contains unified data for simulation
type GlobalState struct {
	Price        *uint256.Int `json:"price"`
	Tick         int32        `json:"tick"`
	LastFee      uint16       `json:"lF"`
	PluginConfig uint8        `json:"pC"`
	CommunityFee uint16       `json:"cF"`
	Unlocked     bool         `json:"un"`
}

type FetchRPCResult struct {
	Liquidity   *big.Int
	State       GlobalState
	TickSpacing *big.Int
	Reserve0    *big.Int
	Reserve1    *big.Int

	Timepoints       map[uint16]Timepoint
	VolatilityOracle VolatilityOraclePlugin
	SlidingFee       SlidingFeeConfig
	DynamicFee       DynamicFeeConfig
	BlockNumber      *big.Int
}

type Timepoint struct {
	Initialized          bool         `json:"init,omitempty"` // whether the timepoint is initialized
	BlockTimestamp       uint32       `json:"ts,omitempty"`   // the block timestamp of the timepoint
	TickCumulative       int64        `json:"cum,omitempty"`  // the tick accumulator, i.e., tick * time elapsed since the pool was first initialized
	VolatilityCumulative *uint256.Int `json:"vo,omitempty"`   // the volatility accumulator; overflow after ~34800 years is desired :)
	Tick                 int32        `json:"tick,omitempty"` // tick at this blockTimestamp
	AverageTick          int32        `json:"avgT,omitempty"` // average tick at this blockTimestamp (for WINDOW seconds)
	WindowStartIndex     uint16       `json:"wsI,omitempty"`  // closest timepoint lte WINDOW seconds ago (or oldest timepoint), should be used only from the last timepoint!
}

// TimepointRPC same as Timepoint but with bigInt for correct deserialization
type TimepointRPC struct {
	Initialized          bool
	BlockTimestamp       uint32
	TickCumulative       *big.Int
	VolatilityCumulative *big.Int
	Tick                 *big.Int
	AverageTick          *big.Int
	WindowStartIndex     uint16
}

type Extra struct {
	Liquidity        *uint256.Int           `json:"liq"`
	GlobalState      GlobalState            `json:"gS"`
	Ticks            []v3Entities.Tick      `json:"ticks"`
	TickSpacing      int32                  `json:"tS"`
	Timepoints       map[uint16]Timepoint   `json:"tP"`
	VolatilityOracle VolatilityOraclePlugin `json:"vo"`
	SlidingFee       SlidingFeeConfig       `json:"sF"`
	DynamicFee       DynamicFeeConfig       `json:"dF"`
}

type VolatilityOraclePlugin struct {
	TimepointIndex         uint16 `json:"tpIdx,omitempty"`
	LastTimepointTimestamp uint32 `json:"lastTs,omitempty"`
	IsInitialized          bool   `json:"init,omitempty"`
}

type DynamicFeeConfig struct {
	Alpha1      uint16 `json:"a1,omitempty"` // max value of the first sigmoid
	Alpha2      uint16 `json:"a2,omitempty"` // max value of the second sigmoid
	Beta1       uint32 `json:"b1,omitempty"` // shift along the x-axis for the first sigmoid
	Beta2       uint32 `json:"b2,omitempty"` // shift along the x-axis for the second sigmoid
	Gamma1      uint16 `json:"g1,omitempty"` // horizontal stretch factor for the first sigmoid
	Gamma2      uint16 `json:"g2,omitempty"` // horizontal stretch factor for the second sigmoid
	VolumeBeta  uint32 `json:"vB,omitempty"` // shift along the x-axis for the outer volume-sigmoid
	VolumeGamma uint16 `json:"vG,omitempty"` // horizontal stretch factor the outer volume-sigmoid
	BaseFee     uint16 `json:"bF,omitempty"` // minimum possible fee
}

type SlidingFeeConfig struct {
	ZeroToOneFeeFactor *uint256.Int `json:"0to1fF,omitempty"`
	OneToZeroFeeFactor *uint256.Int `json:"1to0fF,omitempty"`
}

type StaticExtra struct {
	UseBasePluginV2 bool `json:"pluginV2"`
}

// StateUpdate to be returned instead of updating state when calculating amountOut
type StateUpdate struct {
	Liquidity *uint256.Int
	Price     *uint256.Int
	Tick      int32
}

type PoolMeta struct {
	BlockNumber uint64       `json:"blockNumber"`
	PriceLimit  *uint256.Int `json:"priceLimit"`
}

func (tp *TimepointRPC) toTimepoint() Timepoint {
	volatilityCumulative := uint256.MustFromBig(tp.VolatilityCumulative)
	return Timepoint{
		Initialized:          tp.Initialized,
		BlockTimestamp:       tp.BlockTimestamp,
		TickCumulative:       tp.TickCumulative.Int64(),
		VolatilityCumulative: volatilityCumulative,
		Tick:                 int32(tp.Tick.Int64()),
		AverageTick:          int32(tp.AverageTick.Int64()),
		WindowStartIndex:     tp.WindowStartIndex,
	}
}

type FeesAmount struct {
	communityFeeAmount *uint256.Int
	pluginFeeAmount    *uint256.Int
}

type SwapCalculationCache struct {
	amountRequiredInitial *uint256.Int // The initial value of the exact input/output amount
	amountCalculated      *uint256.Int // The additive amount of total output/input calculated through the swap
	pluginFee             *uint256.Int // The plugin fee
	communityFee          *uint256.Int // The community fee of the selling token, uint256 to minimize casts
	fee                   uint64 // The current fee value in hundredths of a bip, i.e. 1e-6
	exactInput            bool         // Whether the exact input or output is specified
}

type PriceMovementCache struct {
	stepSqrtPrice *uint256.Int // The Q64.96 sqrt of the price at the start of the step, uint256 to minimize casts
	nextTickPrice *uint256.Int // The Q64.96 sqrt of the price calculated from the _nextTick_, uint256 to minimize casts
	input         *uint256.Int // The additive amount of tokens that have been provided
	output        *uint256.Int // The additive amount of token that have been withdrawn
	feeAmount     *uint256.Int // The total amount of fee earned within a current step

	nextTick    int32 // The tick till the current step goes
	initialized bool  // True if the _nextTick is initialized
}
