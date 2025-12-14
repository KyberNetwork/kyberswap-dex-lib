package gateway

import (
	"math/big"
)

// Extra contains the pool state data that gets updated from on-chain
// Note: Only ONE-WAY (synchronous) swaps are supported
type Extra struct {
	IsPaused bool `json:"isPaused"`
	
	// Token supplies
	IUSDSupply       *big.Int `json:"iusdSupply"`       // Total iUSD supply
	SIUSDTotalAssets *big.Int `json:"siusdTotalAssets"` // siUSD vault total assets (iUSD backing)
	SIUSDSupply      *big.Int `json:"siusdSupply"`      // Total siUSD shares
	
	// liUSD token info
	LIUSDSupplies []string `json:"liusdSupplies"` // Total supply for each liUSD token
	
	// Conversion rates
	// Mint: 1:1 (USDC → iUSD, controlled by MintController)
	// Stake: ERC4626 share price (iUSD → siUSD, calculated from totalAssets/totalSupply)
	// Lock: 1:1 (iUSD → liUSD, controlled by LockingController)
}

// Meta contains metadata about the pool state
type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

