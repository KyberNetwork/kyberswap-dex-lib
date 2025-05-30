package algebrav1

import (
	"fmt"
	"math/big"
	"strconv"

	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	v3EntitiesBigInt "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/uniswapv3"
)

type int24 = int32
type int56 = int64

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
	LiquidityTotal           *big.Int
	LiquidityDelta           *big.Int
	OuterFeeGrowth0Token     *big.Int
	OuterFeeGrowth1Token     *big.Int
	OuterTickCumulative      *big.Int
	OuterSecondsPerLiquidity *big.Int
	OuterSecondsSpent        uint32
	Initialized              bool
}

type SubgraphPoolTicks struct {
	ID    string     `json:"id"`
	Ticks []TickResp `json:"ticks"`
}

// for algebra v1 with single fee for both direction
type rpcGlobalStateSingleFee struct {
	Price              *big.Int
	Tick               *big.Int
	Fee                uint16
	TimepointIndex     uint16
	CommunityFeeToken0 uint16
	CommunityFeeToken1 uint16
	Unlocked           bool
}

// rpcGlobalStateDirFee for algebra v1 camelot and similar (directional fee)
type rpcGlobalStateDirFee struct {
	Price              *big.Int
	Tick               *big.Int
	FeeZto             uint16
	FeeOtz             uint16
	TimepointIndex     uint16
	CommunityFeeToken0 uint16
	CommunityFeeToken1 uint16
	Unlocked           bool
}

// GlobalStateUint256 contains unified data for simulation
type GlobalStateUint256 struct {
	Price              *v3Utils.Uint160 `json:"price"`
	Tick               int              `json:"tick"`
	FeeZto             uint16           `json:"feeZto"`
	FeeOtz             uint16           `json:"feeOtz"`
	TimepointIndex     uint16           `json:"timepoint_index"`
	CommunityFeeToken0 uint16           `json:"community_fee_token0"`
	CommunityFeeToken1 uint16           `json:"community_fee_token1"`
	Unlocked           bool             `json:"unlocked"`
}

// GlobalState contains unified data for simulation
type GlobalState struct {
	Price              *big.Int `json:"price"`
	Tick               *big.Int `json:"tick"`
	FeeZto             uint16   `json:"feeZto"`
	FeeOtz             uint16   `json:"feeOtz"`
	TimepointIndex     uint16   `json:"timepoint_index"`
	CommunityFeeToken0 uint16   `json:"community_fee_token0"`
	CommunityFeeToken1 uint16   `json:"community_fee_token1"`
	Unlocked           bool     `json:"unlocked"`
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

type TimepointUint256 struct {
	Initialized                   bool         `json:"initialized"`                   // whether or not the timepoint is initialized
	BlockTimestamp                uint32       `json:"blockTimestamp"`                // the block timestamp of the timepoint
	AverageTick                   int24        `json:"averageTick"`                   // average tick at this blockTimestamp
	TickCumulative                int56        `json:"tickCumulative"`                // the tick accumulator, i.e. tick * time elapsed since the pool was first initialized
	SecondsPerLiquidityCumulative *uint256.Int `json:"secondsPerLiquidityCumulative"` // the seconds per liquidity since the pool was first initialized
	VolatilityCumulative          *uint256.Int `json:"volatilityCumulative"`          // the volatility accumulator; overflow after ~34800 years is desired :)
	VolumePerLiquidityCumulative  *uint256.Int `json:"volumePerLiquidityCumulative"`  // the gmean(volumes)/liquidity accumulator
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

func (t Timepoint) GetInitialized() bool {
	return t.Initialized
}

func (t Timepoint) GetBlockTimestamp() uint32 {
	return t.BlockTimestamp
}

func (tp TimepointRPC) GetInitialized() bool {
	return tp.Initialized
}

func (tp TimepointRPC) GetBlockTimestamp() uint32 {
	return tp.BlockTimestamp
}

func (tp TimepointRPC) ToTimepoint() Timepoint {
	return Timepoint{
		Initialized:                   tp.Initialized,
		BlockTimestamp:                tp.BlockTimestamp,
		TickCumulative:                tp.TickCumulative.Int64(),
		SecondsPerLiquidityCumulative: tp.SecondsPerLiquidityCumulative,
		VolatilityCumulative:          tp.VolatilityCumulative,
		AverageTick:                   int24(tp.AverageTick.Int64()),
		VolumePerLiquidityCumulative:  tp.VolumePerLiquidityCumulative,
	}
}

// TimepointRPC is same as Timepoint but with bigInt for correct deserialization
type TimepointRPC struct {
	Initialized                   bool
	BlockTimestamp                uint32
	TickCumulative                *big.Int
	SecondsPerLiquidityCumulative *big.Int
	VolatilityCumulative          *big.Int
	AverageTick                   *big.Int
	VolumePerLiquidityCumulative  *big.Int
}

type ExtraUint256 struct {
	Liquidity   *uint256.Int       `json:"liquidity"`
	GlobalState GlobalStateUint256 `json:"globalState"`
	Ticks       []v3Entities.Tick  `json:"ticks"`
	TickSpacing int24              `json:"tickSpacing"`
}

type Extra struct {
	Liquidity   *big.Int                `json:"liquidity"`
	GlobalState GlobalState             `json:"globalState"`
	Ticks       []v3EntitiesBigInt.Tick `json:"ticks"`
	TickSpacing int24                   `json:"tickSpacing"`
}

// StateUpdate should be returned instead since we won't update the state when calculating amountOut
type StateUpdate struct {
	Liquidity   *uint256.Int
	GlobalState GlobalStateUint256
}

// StateUpdateBigInt should be returned instead since we won't update the state when calculating amountOut
type StateUpdateBigInt struct {
	Liquidity   *big.Int
	GlobalState GlobalState
}

type PoolMeta = uniswapv3.PoolMeta

func transformTickRespToTickBigInt(tickResp TickResp) (v3EntitiesBigInt.Tick, error) {
	liquidityGross, ok := new(big.Int).SetString(tickResp.LiquidityGross, 10)
	if !ok {
		return v3EntitiesBigInt.Tick{}, fmt.Errorf("can not convert liquidityGross string to bigInt, tick: %v",
			tickResp.TickIdx)
	}

	liquidityNet, ok := new(big.Int).SetString(tickResp.LiquidityNet, 10)
	if !ok {
		return v3EntitiesBigInt.Tick{}, fmt.Errorf("can not convert liquidityNet string to bigInt, tick: %v",
			tickResp.TickIdx)
	}

	tickIdx, err := strconv.Atoi(tickResp.TickIdx)
	if err != nil {
		return v3EntitiesBigInt.Tick{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", tickResp.TickIdx)
	}

	return v3EntitiesBigInt.Tick{
		Index:          tickIdx,
		LiquidityGross: liquidityGross,
		LiquidityNet:   liquidityNet,
	}, nil
}
