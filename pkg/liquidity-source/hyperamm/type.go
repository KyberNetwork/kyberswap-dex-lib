package hyperamm

import (
	"github.com/holiman/uint256"
)

// StaticExtra holds immutable per-pool data that changes only when a pool is
// redeployed.  It is stored once during pool discovery and never refreshed.
type StaticExtra struct {
	// SwapFeeModule is the per-pool HyperAMMSwapFeeModule contract address.
	SwapFeeModule string `json:"s"`
}

// Extra holds mutable per-pool state that is refreshed by the pool tracker on
// every poll cycle.
type Extra struct {
	// FairPriceFrom is the oracle-derived price (in 1e18 scale) for swapping from token i
	// A value P means: 1 wei of token i in gives P/1e18 wei of token 1-i out before fees.
	FairPriceFrom [2]*uint256.Int `json:"r"`
	// RefFeeFrom is the total effective fee in bps for a reference 1-unit from token i swap,
	// as returned by getSwapFeeInBips. It already captures base fee + imbalance fee + current premium adjustment.
	RefFeeFrom [2]uint64 `json:"f"`
	// IsPaused mirrors the HyperAMM.paused() flag.
	IsPaused bool `json:"p,omitempty"`
}

// MetaInfo is returned by GetMetaInfo and consumed by the transaction builder.
type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	BlockNumber     uint64 `json:"bN,omitempty"`
	IsZeroToOne     bool   `json:"0,omitempty"`
}
