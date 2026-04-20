package atokenswap

import (
	"github.com/holiman/uint256"
)

// OutputTokenState represents state for a single output token
type OutputTokenState struct {
	RateWithPremium    *uint256.Int `json:"r"` // Rate with premium applied
	AvailableLiquidity *uint256.Int `json:"l"` // Available liquidity
	MaxSwap            *uint256.Int `json:"m"` // Maximum swap amount
}

// Extra represents the essential ATokenSwap pool data
type Extra struct {
	Paused       bool               `json:"p,omitempty"`
	OutputStates []OutputTokenState `json:"o"` // State for each output token
}

// SwapInfo contains information passed during swap execution
type SwapInfo struct {
	NewPoolState *Extra `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber,omitempty"`
}
