package parityprop

import "github.com/holiman/uint256"

// StaticExtra is immutable per-pool metadata set once at discovery time.
type StaticExtra struct {
	Base       string `json:"base"`
	Quote      string `json:"quote"`
	BaseScale  string `json:"baseScale"`  // 10**base.decimals(), decimal string
	QuoteScale string `json:"quoteScale"` // 10**quote.decimals(), decimal string
}

// Extra is rewritten end-to-end on each tracker refresh. Every field here
// mirrors a guard PmmPool._quote() checks on-chain.
type Extra struct {
	Paused bool `json:"paused"`

	FeeBps           uint16       `json:"feeBps"`
	MaxSpreadBps     uint16       `json:"maxSpreadBps"`
	MaxStaleness     uint32       `json:"maxStaleness"`
	MaxSwapNotional  *uint256.Int `json:"maxSwapNotional"`
	MaxBlockNotional *uint256.Int `json:"maxBlockNotional"`
	MinBaseReserve   *uint256.Int `json:"minBaseReserve"`
	MinQuoteReserve  *uint256.Int `json:"minQuoteReserve"`

	Oracle          string `json:"oracle"`
	OracleBid       uint64 `json:"oracleBid"`
	OracleMid       uint64 `json:"oracleMid"`
	OracleAsk       uint64 `json:"oracleAsk"`
	OracleUpdatedAt uint64 `json:"oracleUpdatedAt"`

	// LastSwapBlock / BlockNotional are the on-chain per-block notional
	// accounting PmmPool.swap() maintains; needed to replicate
	// BlockCapExceeded within the same block as prior simulated swaps.
	LastSwapBlock uint64       `json:"lastSwapBlock"`
	BlockNotional *uint256.Int `json:"blockNotional"`
}

// SwapInfo is the per-swap result UpdateBalance consumes.
type SwapInfo struct {
	Notional      *uint256.Int `json:"notional"`
	SwapBlock     uint64       `json:"swapBlock"`
	BlockNotional *uint256.Int `json:"blockNotional"` // accumulated notional for SwapBlock after this swap
}

type MetaInfo struct {
	BlockNumber uint64 `json:"blockNumber"`
}
