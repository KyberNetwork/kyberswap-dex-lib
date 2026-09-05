package dualpool

import (
	"math/big"

	"github.com/holiman/uint256"
)

// Bucket is one entry of the hook's per-pool liquidity distribution: a tick range
// and the share of the reserves (in basis points, summing to 10 000) deployed into it.
type Bucket struct {
	TickLower int32  `json:"l"`
	TickUpper int32  `json:"u"`
	WeightBps uint16 `json:"w"`
}

// Extra is the hook state persisted in the pool's HookExtra by Track and read by
// the simulator.
type Extra struct {
	Live         bool         `json:"live"`
	Balance0     *uint256.Int `json:"b0"` // getEffectiveLiquidity: what the JIT cycle can deploy right now
	Balance1     *uint256.Int `json:"b1"`
	SqrtPriceX96 *uint256.Int `json:"sP"`
	Tick         int32        `json:"t"`
	ProtocolFee  uint32       `json:"pF"` // v4 packed protocol fee from slot0
	LpFee        uint32       `json:"lF"` // static key fee, as slot0 reports it
	Buckets      []Bucket     `json:"bk"`
}

// SwapInfo carries the post-swap state from CalcAmountOut to UpdateBalance.
type SwapInfo struct {
	ZeroForOne   bool
	AmountIn     *uint256.Int
	AmountOut    *uint256.Int
	SqrtPriceX96 *uint256.Int
	Tick         int32
}

type distributionEntry struct {
	TickLower *big.Int
	TickUpper *big.Int
	WeightBps uint16
}

// multi-output ABI results unpack into one struct whose fields are the output names
type effectiveLiquidityResp struct {
	Token0 *big.Int
	Token1 *big.Int
}

type slot0Resp struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	ProtocolFee  *big.Int
	LpFee        *big.Int
}
