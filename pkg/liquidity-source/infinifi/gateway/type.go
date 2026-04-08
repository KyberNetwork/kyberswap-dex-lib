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
	LIUSDBuckets []bucket `json:"liusdBuckets"`
}

// Meta contains metadata about the pool state
type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Action int

const (
	ActionMint           Action = iota // 0: mint => USDC → iUSD
	ActionRedeem                       // 1: redeem => iUSD → USDC
	ActionStake                        // 2: stake => iUSD → siUSD
	ActionUnstake                      // 3: unstake => siUSD → iUSD
	ActionMintAndStake                 // 4: mint and stake => USDC → siUSD
	ActionCreatePosition               // 5: create position => iUSD → liUSD
	ActionMintAndLock                  // 6: mint and lock => USDC → liUSD
)

type SwapInfo struct {
	Action          Action `json:"action"`
	UnwindingEpochs int    `json:"unwindingEpochs"`
}
