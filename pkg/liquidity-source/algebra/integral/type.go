package integral

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/KyberNetwork/int256"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
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

func (t TickResp) transformTickRespToTick() (v3Entities.Tick, error) {
	liquidityGross := new(big.Int)
	liquidityGross, ok := liquidityGross.SetString(t.LiquidityGross, 10)
	if !ok {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", t.TickIdx)
	}

	liquidityNet := new(big.Int)
	liquidityNet, ok = liquidityNet.SetString(t.LiquidityNet, 10)
	if !ok {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", t.TickIdx)
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

// for algebra v1 with single fee for both direction
type GlobalStateFromRPC struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}

// unified data for simulation
type GlobalState struct {
	Price        *uint256.Int `json:"price"`
	Tick         int32        `json:"tick"`
	LastFee      uint16       `json:"lastFee"`
	PluginConfig uint8        `json:"pluginConfig"`
	CommunityFee uint16       `json:"communityFee"`
	Unlocked     bool         `json:"unlocked"`
}

type FeeConfiguration struct {
	Alpha1      uint16 `json:"alpha1"`      // max value of the first sigmoid
	Alpha2      uint16 `json:"alpha2"`      // max value of the second sigmoid
	Beta1       uint32 `json:"beta1"`       // shift along the x-axis for the first sigmoid
	Beta2       uint32 `json:"beta2"`       // shift along the x-axis for the second sigmoid
	Gamma1      uint16 `json:"gamma1"`      // horizontal stretch factor for the first sigmoid
	Gamma2      uint16 `json:"gamma2"`      // horizontal stretch factor for the second sigmoid
	VolumeBeta  uint32 `json:"volumeBeta"`  // shift along the x-axis for the outer volume-sigmoid
	VolumeGamma uint16 `json:"volumeGamma"` // horizontal stretch factor the outer volume-sigmoid
	BaseFee     uint16 `json:"baseFee"`     // minimum possible fee
}

type FetchRPCResult struct {
	Liquidity   *big.Int    `json:"liquidity"`
	State       GlobalState `json:"state"`
	TickSpacing *big.Int    `json:"tickSpacing"`
	Reserve0    *big.Int    `json:"reserve0"`
	Reserve1    *big.Int    `json:"reserve1"`

	Timepoints       map[uint16]Timepoint
	VotatilityOracle VotatilityOraclePlugin
	SlidingFee       SlidingFeePlugin
	DynamicFee       DynamicFeePlugin
	BlockNumber      *big.Int
}

type Timepoint struct {
	Initialized          bool         // whether or not the timepoint is initialized
	BlockTimestamp       uint32       // the block timestamp of the timepoint
	TickCumulative       int64        // the tick accumulator, i.e., tick * time elapsed since the pool was first initialized
	VolatilityCumulative *uint256.Int // the volatility accumulator; overflow after ~34800 years is desired :)
	Tick                 int32        // tick at this blockTimestamp
	AverageTick          int32        // average tick at this blockTimestamp (for WINDOW seconds)
	WindowStartIndex     uint16       // closest timepoint lte WINDOW seconds ago (or oldest timepoint), should be used only from the last timepoint!
}

// same as Timepoint but with bigInt for correct deserialization
type TimepointRPC struct {
	Initialized          bool
	BlockTimestamp       uint32
	TickCumulative       *big.Int
	VolatilityCumulative *big.Int
	Tick                 *big.Int
	AverageTick          *big.Int
	WindowStartIndex     uint16
}

type StaticExtra struct {
	UseBasePluginV2 bool `json:"useBasePluginV2"`
}

type Extra struct {
	Liquidity        *big.Int               `json:"liquidity"`
	GlobalState      GlobalState            `json:"globalState"`
	Ticks            []v3Entities.Tick      `json:"ticks"`
	TickSpacing      int32                  `json:"tickSpacing"`
	Timepoints       map[uint16]Timepoint   `json:"timepoints"`
	VotatilityOracle VotatilityOraclePlugin `json:"votalityOracle"`
	SlidingFee       SlidingFeePlugin       `json:"slidingFee"`
	DynamicFee       DynamicFeePlugin       `json:"dynamicFee"`
}

// we won't update the state when calculating amountOut, return this struct instead
type StateUpdate struct {
	Liquidity   *uint256.Int
	GlobalState GlobalState
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
	communityFee          *uint256.Int // The community fee of the selling token, uint256 to minimize casts
	amountRequiredInitial *int256.Int  // The initial value of the exact input/output amount
	amountCalculated      *int256.Int  // The additive amount of total output/input calculated through the swap
	exactInput            bool         // Whether the exact input or output is specified
	fee                   uint32       // The current fee value in hundredths of a bip, i.e. 1e-6
	pluginFee             uint32       // The plugin fee
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

type FeeFactors struct {
	ZeroToOneFeeFactor *uint256.Int
	OneToZeroFeeFactor *uint256.Int
}
