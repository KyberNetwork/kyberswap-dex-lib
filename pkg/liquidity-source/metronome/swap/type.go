package metronomeswap

import (
	"github.com/holiman/uint256"
)

// StaticExtra holds data that never changes for a given pool after discovery.
type StaticExtra struct {
	PoolRegistry string `json:"poolRegistry"`
}

// TokenState is the per-synthetic-token state a swap leg needs. There are no pooled
// reserves in Metronome — capacity is bounded by MaxTotalSupply-TotalSupply headroom
// on the *output* leg (see docs/explorer.md's shared_vault notes for why).
type TokenState struct {
	DebtToken      string       `json:"debtToken"`
	IsActive       bool         `json:"isActive"`
	MaxTotalSupply *uint256.Int `json:"maxTotalSupply"`
	TotalSupply    *uint256.Int `json:"totalSupply"`
	PriceInUsd     *uint256.Int `json:"priceInUsd"` // MasterOracle.getPriceInUsd(token), 1e18-scaled
}

// Extra is the per-pool mutable state pool_simulator.go consumes. Populated and kept
// fresh by pool_tracker.go.
type Extra struct {
	// SwapActive is the AND of Pool.isSwapActive, Pool.paused, Pool.everythingStopped,
	// PoolRegistry.paused, PoolRegistry.everythingStopped — collapsed to one flag so the
	// simulator doesn't need to know about the multi-level pause hierarchy.
	SwapActive bool `json:"swapActive"`

	FeeProvider  string `json:"feeProvider"`
	MasterOracle string `json:"masterOracle"`

	// Tokens is keyed by synthetic token address (== entity.Pool.Tokens[i]).
	Tokens map[string]TokenState `json:"tokens"`

	// SwapFeesBps is keyed by "<tokenIn>-<tokenOut>" (ordered pair — NOT symmetric).
	// Value is FeeProvider.swapFees(tokenIn, tokenOut), 1e18-scaled fraction (e.g. 45bps == 4500000000000000).
	SwapFeesBps map[string]*uint256.Int `json:"swapFeesBps"`
}
