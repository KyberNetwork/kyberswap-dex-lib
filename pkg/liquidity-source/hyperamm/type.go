package hyperamm

import "github.com/holiman/uint256"

// StaticExtra holds immutable per-pool data that changes only when a pool is
// redeployed.  It is stored once during pool discovery and never refreshed.
type StaticExtra struct {
	// SwapFeeModule is the per-pool HyperAMMSwapFeeModule contract address.
	SwapFeeModule string `json:"swapFeeModule"`
	// IsToken0Based indicates whether the pool's LP value is denominated in
	// token0 (true) or token1 (false).
	IsToken0Based bool `json:"isToken0Based"`
}

// Extra holds mutable per-pool state that is refreshed by the pool tracker on
// every poll cycle.
type Extra struct {
	// FairPrice0To1 is the oracle-derived price (in 1e18 scale) for swapping
	// token0 → token1.  A value P means: 1 wei of token0 in gives P/1e18 wei
	// of token1 out before fees.
	FairPrice0To1 *uint256.Int `json:"fp01"`
	// FairPrice1To0 is the oracle-derived price for token1 → token0.
	FairPrice1To0 *uint256.Int `json:"fp10"`
	// BaseFeeBps is the minimum flat fee charged on every swap, in basis
	// points (10 000 = 100 %).
	BaseFeeBps uint16 `json:"baseFeeBps"`
	// RefFee0To1 is the total effective fee in bps for a reference 1-unit
	// token0 → token1 swap, as returned by previewSwapFeeInBips.  It already
	// captures base fee + imbalance fee + current premium adjustment.
	RefFee0To1 uint64 `json:"refFee01"`
	// RefFee1To0 is the equivalent for token1 → token0.
	RefFee1To0 uint64 `json:"refFee10"`
	// IsPaused mirrors the HyperAMM.paused() flag.
	IsPaused bool `json:"isPaused"`
}

// SwapInfo is passed from CalcAmountOut to UpdateBalance.
type SwapInfo struct {
	IsZeroToOne bool `json:"isZeroToOne"`
}

// MetaInfo is returned by GetMetaInfo and consumed by the transaction builder.
type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
	IsZeroToOne bool   `json:"isZeroToOne"`
}
