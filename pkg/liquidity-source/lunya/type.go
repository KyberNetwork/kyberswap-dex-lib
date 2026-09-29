package lunya

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

type Gas struct {
	BaseGas          int64
	CrossInitTickGas int64
}

// Extra is a pool's swap state, all read at one block. The fields below the common ones belong to one
// pool type or the other: ticks to CL and CP, the curve to STABLE.
type Extra struct {
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	Liquidity    *uint256.Int `json:"liquidity"`
	// Fee is what the next swap is charged, in pips: the plugin's currentFee() when the pool has
	// DYNAMIC_FEE on, slot0.fee otherwise.
	Fee      uint32 `json:"fee"`
	FeeToken uint8  `json:"feeToken"`
	// Halted is set when the pool's plugin would revert beforeSwap (SecurityModule status not Active).
	Halted bool `json:"halted,omitempty"`

	// CL and CP: the current tick, and the initialized ticks ascending
	Tick  int    `json:"tick,omitempty"`
	Ticks []Tick `json:"ticks,omitempty"`

	// STABLE: the amounts on the curve in raw units - not the balances, which also hold uncollected
	// fees - the multipliers lifting them to 18 decimals, and sqrt(rate1/rate0) in Q64.96
	CurveReserve0     *uint256.Int `json:"curveReserve0,omitempty"`
	CurveReserve1     *uint256.Int `json:"curveReserve1,omitempty"`
	Rate0             *uint256.Int `json:"rate0,omitempty"`
	Rate1             *uint256.Int `json:"rate1,omitempty"`
	PriceScaleSqrtQ96 *uint256.Int `json:"priceScaleSqrtQ96,omitempty"`
	// AmplificationX100 is A in hundredths at the block read; Ramp is set while A is moving, so the
	// simulator can place A at the time it prices.
	AmplificationX100 uint32             `json:"amplificationX100,omitempty"`
	Ramp              *AmplificationRamp `json:"ramp,omitempty"`
}

type Tick struct {
	Index          int          `json:"index"`
	LiquidityGross *uint256.Int `json:"liquidityGross"`
	LiquidityNet   *int256.Int  `json:"liquidityNet"`
}

type AmplificationRamp struct {
	StartAmplification  uint32 `json:"startAmplification"`
	TargetAmplification uint32 `json:"targetAmplification"`
	StartTime           uint32 `json:"startTime"`
	EndTime             uint32 `json:"endTime"`
}

// StaticExtra carries the pool type, which the log that created the pool already named.
type StaticExtra struct {
	PoolType uint8 `json:"poolType"`
}

type SwapInfo struct {
	RemainingAmountIn     *uint256.Int `json:"rAI,omitempty"`
	NextStateSqrtRatioX96 *uint256.Int `json:"nSqrtRx96"`
	// CL and CP
	NextStateLiquidity   *uint256.Int `json:"-"`
	NextStateTickCurrent int          `json:"nT,omitempty"`
	// STABLE
	NextCurveReserve0 *uint256.Int `json:"-"`
	NextCurveReserve1 *uint256.Int `json:"-"`
}

// PoolMeta carries what the adapter needs besides the pool address: the sqrtPriceLimitX96 the simulator
// priced with, so on-chain execution stops where the quote did.
type PoolMeta struct {
	PriceLimit  *uint256.Int `json:"priceLimit"`
	BlockNumber uint64       `json:"blockNumber"`
}

type slot0Resp struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	FeeProtocol0 uint16
	FeeProtocol1 uint16
}

type tickResp struct {
	LiquidityGross        *big.Int
	LiquidityNet          *big.Int
	FeeGrowthOutside0X128 *big.Int
	FeeGrowthOutside1X128 *big.Int
	Initialized           bool
}

type amplificationRampResp struct {
	StartAmplification  uint32
	TargetAmplification uint32
	StartTime           uint32
	EndTime             uint32
}
