package pancakeprop

import (
	"github.com/holiman/uint256"
)

type Config struct {
	DexID string `json:"dexID"`
	// Entry is the PropAMMEntry; pricing and vault are read from it.
	Entry string `json:"entry"`
	// BlockTimeMs converts the price's expiry block into wall-clock time.
	BlockTimeMs uint64 `json:"blockTimeMs"`
	// PricingImplementations are the audited implementations behind the pricing proxy. The simulator
	// ports their bytecode, so the tracker rejects any other (upgraded) implementation.
	PricingImplementations []string `json:"pricingImplementations"`
}

// StaticExtra is immutable per-pair metadata set at discovery.
type StaticExtra struct {
	Entry   string `json:"entry"`
	Pricing string `json:"pricing"`
	Vault   string `json:"vault"`
	PairID  uint64 `json:"pairId"`
}

// Extra is the mutable pair state, rewritten on every tracker refresh.
type Extra struct {
	Paused bool `json:"paused"`
	// Packed is the MM's current price word (mid, spreads, depth curve, expiry block).
	Packed uint256.Int `json:"packed"`
	// MaxInput is the pair's per-epoch capacity ceiling per input token (token0, token1).
	MaxInput [2]uint256.Int `json:"maxInput"`
	// Cum is the cumulative input per input token in the current price epoch (token0, token1).
	Cum [2]uint256.Int `json:"cum"`
	// ExpiredAtMs is the wall-clock time (unix ms) after which the price reverts as stale.
	ExpiredAtMs uint64 `json:"expiredAtMs"`
}

// PoolMeta is handed to aggregator-encoding.
type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
	Entry       string `json:"entry"`
}

type SwapInfo struct {
	IndexIn   int
	AmountIn  uint256.Int
	AmountOut uint256.Int
}
