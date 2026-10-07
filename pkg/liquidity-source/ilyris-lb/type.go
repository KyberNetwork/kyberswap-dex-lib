package ilyrislb

import "github.com/holiman/uint256"

// StaticExtra holds pool immutables.
type StaticExtra struct {
	BinStepBps uint32 `json:"binStepBps"`
	DecimalsX  uint8  `json:"decimalsX"`
	DecimalsY  uint8  `json:"decimalsY"`
}

// Extra is the mutable snapshot, all read at the entity's BlockNumber.
type Extra struct {
	ActiveID int32      `json:"activeId"`
	Bins     []Bin      `json:"bins"` // ascending by ID, populated bins only
	Fee      *FeeParams `json:"fee"`
	// Market guard state. Quotes skip the guard on chain but swaps do not.
	GuardSwapsPaused bool   `json:"guardSwapsPaused,omitempty"`
	GuardFreezeEnd   uint64 `json:"guardFreezeEnd,omitempty"`
	BlockTimestamp   uint64 `json:"blockTimestamp"`
}

// Bin is one price level. Reserves are uint128 on chain and travel as decimal strings.
type Bin struct {
	ID       int32       `json:"id"`
	ReserveX uint256.Int `json:"x"`
	ReserveY uint256.Int `json:"y"`
}

// FeeParams is BinPool.feeConfig() plus feeState(): what getTotalFeeRate() reads besides
// binStepBps, activeId and block.timestamp.
type FeeParams struct {
	BaseFactor               uint32 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint32 `json:"variableFeeControl"`
	MaxVolatilityAccumulator uint32 `json:"maxVolatilityAccumulator"`

	VolatilityAccumulator uint32 `json:"volatilityAccumulator"`
	VolatilityReference   uint32 `json:"volatilityReference"`
	IDReference           int32  `json:"idReference"`
	TimeLastUpdate        uint64 `json:"timeLastUpdate"`
}

// Metadata is the lister cursor: an offset into BinFactory.allPools, which is append-only.
type Metadata struct {
	Offset int `json:"offset"`
}

// SwapInfo hands the per-bin fills from CalcAmountOut to UpdateBalance.
type SwapInfo struct {
	xForY       bool
	newActiveID int32
	fills       []binFill
}

// binFill is the net (post-fee) movement of one bin: in joins the input side, out leaves.
type binFill struct {
	idx     int
	in, out uint256.Int
}
