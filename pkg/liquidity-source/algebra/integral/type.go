package integral

import (
	"fmt"
	"math/big"
	"strconv"

	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
)

type int24 = int32
type int56 = int64

type uint24 = uint32
type uint56 = uint64

type PluginFee struct {
	OverrideFee uint24
	PluginFee   uint24
}

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

type SubgraphPoolTicks struct {
	ID    string     `json:"id"`
	Ticks []TickResp `json:"ticks"`
}

// for algebra v1 with single fee for both direction
type rpcGlobalStateSingleFee struct {
	Price        *big.Int
	Tick         *big.Int
	LastFee      uint16
	PluginConfig uint8
	CommunityFee uint16
	Unlocked     bool
}

// for algebra v1 camelot and similar (directional fee)
type rpcGlobalStateDirFee struct {
	Price              *big.Int
	Tick               *big.Int
	FeeZto             uint16
	FeeOtz             uint16
	TimepointIndex     uint16
	CommunityFeeToken0 uint8
	CommunityFeeToken1 uint8
	Unlocked           bool
}

// unified data for simulation
type GlobalState struct {
	OverrideFee  *big.Int `json:"overrideFee"`
	PluginFee    *big.Int `json:"pluginFee"`
	Price        *big.Int `json:"price"`
	Tick         int32    `json:"tick"`
	LastFee      uint16   `json:"lastFee"`
	PluginConfig uint8    `json:"pluginConfig"`
	CommunityFee uint16   `json:"communityFee"`
	Unlocked     bool     `json:"unlocked"`
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
}

// type Timepoint struct {
// 	Initialized                   bool     `json:"initialized"`                   // whether or not the timepoint is initialized
// 	BlockTimestamp                uint32   `json:"blockTimestamp"`                // the block timestamp of the timepoint
// 	TickCumulative                int56    `json:"tickCumulative"`                // the tick accumulator, i.e. tick * time elapsed since the pool was first initialized
// 	SecondsPerLiquidityCumulative *big.Int `json:"secondsPerLiquidityCumulative"` // the seconds per liquidity since the pool was first initialized
// 	VolatilityCumulative          *big.Int `json:"volatilityCumulative"`          // the volatility accumulator; overflow after ~34800 years is desired :)
// 	AverageTick                   int24    `json:"averageTick"`                   // average tick at this blockTimestamp
// 	VolumePerLiquidityCumulative  *big.Int `json:"volumePerLiquidityCumulative"`  // the gmean(volumes)/liquidity accumulator
// }

type Timepoint struct {
	Initialized          bool     // whether or not the timepoint is initialized
	BlockTimestamp       uint32   // the block timestamp of the timepoint
	TickCumulative       int64    // the tick accumulator, i.e., tick * time elapsed since the pool was first initialized
	VolatilityCumulative *big.Int // the volatility accumulator; overflow after ~34800 years is desired :)
	Tick                 int32    // tick at this blockTimestamp
	AverageTick          int32    // average tick at this blockTimestamp (for WINDOW seconds)
	WindowStartIndex     uint16   // closest timepoint lte WINDOW seconds ago (or oldest timepoint), should be used only from the last timepoint!
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

type Extra struct {
	Liquidity   *big.Int          `json:"liquidity"`
	GlobalState GlobalState       `json:"globalState"`
	Ticks       []v3Entities.Tick `json:"ticks"`
	TickSpacing int24             `json:"tickSpacing"`
}

// we won't update the state when calculating amountOut, return this struct instead
type StateUpdate struct {
	Liquidity   *big.Int
	GlobalState GlobalState
}

type PoolMeta struct {
	BlockNumber uint64   `json:"blockNumber"`
	PriceLimit  *big.Int `json:"priceLimit"`
}

func transformTickRespToTick(tickResp TickResp) (v3Entities.Tick, error) {
	liquidityGross := new(big.Int)
	liquidityGross, ok := liquidityGross.SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v", tickResp.TickIdx)
	}

	liquidityNet := new(big.Int)
	liquidityNet, ok = liquidityNet.SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return v3Entities.Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v", tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return v3Entities.Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return v3Entities.Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}

func (tp *TimepointRPC) toTimepoint() Timepoint {
	return Timepoint{
		Initialized:          tp.Initialized,
		BlockTimestamp:       tp.BlockTimestamp,
		TickCumulative:       tp.TickCumulative.Int64(),
		VolatilityCumulative: tp.VolatilityCumulative,
		Tick:                 int32(tp.Tick.Int64()),
		AverageTick:          int24(tp.AverageTick.Int64()),
		WindowStartIndex:     tp.WindowStartIndex,
	}
}

type FeesAmount struct {
	communityFeeAmount *big.Int
	pluginFeeAmount    *big.Int
}

type SwapCalculationCache struct {
	communityFee          *big.Int // The community fee of the selling token, uint256 to minimize casts
	crossedAnyTick        bool     // If we have already crossed at least one active tick
	amountRequiredInitial *big.Int // The initial value of the exact input/output amount
	amountCalculated      *big.Int // The additive amount of total output/input calculated through the swap
	totalFeeGrowthInput   *big.Int // The initial totalFeeGrowth + the fee growth during a swap
	totalFeeGrowthOutput  *big.Int // The initial totalFeeGrowth for output token, should not change during swap
	exactInput            bool     // Whether the exact input or output is specified
	fee                   uint32   // The current fee value in hundredths of a bip, i.e. 1e-6
	prevInitializedTick   int32    // The previous initialized tick in linked list
	nextInitializedTick   int32    // The next initialized tick in linked list
	pluginFee             uint32   // The plugin fee
}

type PriceMovementCache struct {
	stepSqrtPrice *big.Int // The Q64.96 sqrt of the price at the start of the step, uint256 to minimize casts
	nextTickPrice *big.Int // The Q64.96 sqrt of the price calculated from the _nextTick_, uint256 to minimize casts
	input         *big.Int // The additive amount of tokens that have been provided
	output        *big.Int // The additive amount of token that have been withdrawn
	feeAmount     *big.Int // The total amount of fee earned within a current step
}

type SwapEventParams struct {
	currentPrice     *big.Int
	currentTick      int32
	currentLiquidity *big.Int
}
