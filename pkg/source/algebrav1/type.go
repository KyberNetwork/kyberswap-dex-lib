package algebrav1

import (
	"fmt"
	"math/big"
	"strconv"

	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
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
	Unlocked           bool     `json:"-"`
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
	liquidity                 *big.Int
	state                     GlobalState
	feeConf                   FeeConfiguration
	tickSpacing               *big.Int
	reserve0                  *big.Int
	reserve1                  *big.Int
	timepoints                map[uint16]Timepoint
	volumePerLiquidityInBlock *big.Int
}

type Timepoint struct {
	Initialized                   bool     `json:"initialized"`                   // whether or not the timepoint is initialized
	BlockTimestamp                uint32   `json:"blockTimestamp"`                // the block timestamp of the timepoint
	TickCumulative                int56    `json:"tickCumulative"`                // the tick accumulator, i.e. tick * time elapsed since the pool was first initialized
	SecondsPerLiquidityCumulative *big.Int `json:"secondsPerLiquidityCumulative"` // the seconds per liquidity since the pool was first initialized
	VolatilityCumulative          *big.Int `json:"volatilityCumulative"`          // the volatility accumulator; overflow after ~34800 years is desired :)
	AverageTick                   int24    `json:"averageTick"`                   // average tick at this blockTimestamp
	VolumePerLiquidityCumulative  *big.Int `json:"volumePerLiquidityCumulative"`  // the gmean(volumes)/liquidity accumulator
}

// same as Timepoint but with bigInt for correct deserialization
type TimepointRPC struct {
	Initialized                   bool
	BlockTimestamp                uint32
	TickCumulative                *big.Int
	SecondsPerLiquidityCumulative *big.Int
	VolatilityCumulative          *big.Int
	AverageTick                   *big.Int
	VolumePerLiquidityCumulative  *big.Int
}

type Extra struct {
	Liquidity                 *big.Int             `json:"liquidity"`
	VolumePerLiquidityInBlock *big.Int             `json:"volumePerLiquidityInBlock"`
	GlobalState               GlobalState          `json:"globalState"`
	FeeConfig                 FeeConfiguration     `json:"feeConfig"`
	Ticks                     []v3Entities.Tick    `json:"ticks"`
	TickSpacing               int24                `json:"tickSpacing"`
	Timepoints                map[uint16]Timepoint `json:"timepoints"`
}

// we won't update the state when calculating amountOut, return this struct instead
type StateUpdate struct {
	Liquidity                 *big.Int
	VolumePerLiquidityInBlock *big.Int
	GlobalState               GlobalState
	NewTimepoints             map[uint16]Timepoint
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
