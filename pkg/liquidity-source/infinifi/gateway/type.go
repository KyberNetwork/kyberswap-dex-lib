package gateway

import (
	"math/big"
)

// Extra contains the pool state data that gets updated from on-chain
type Extra struct {
	IsPaused bool `json:"isPaused"`

	// Token supplies
	IUSDSupply       *big.Int `json:"iusdSupply"`       // Total iUSD supply
	SIUSDTotalAssets *big.Int `json:"siusdTotalAssets"` // siUSD vault total assets (iUSD backing)
	SIUSDSupply      *big.Int `json:"siusdSupply"`      // Total siUSD shares

	// liUSD bucket data - each bucket has its own exchange rate
	LIUSDSupplies      []string `json:"liusdSupplies"`      // Total shares for each liUSD bucket
	LIUSDTotalReceipts []string `json:"liusdTotalReceipts"` // Total iUSD locked in each bucket
}

// Meta contains metadata about the pool state
type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
