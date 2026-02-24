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
	ActionMint           Action = iota // 0: mint
	ActionRedeem                       // 1: redeem
	ActionStake                        // 2: stake
	ActionUnstake                      // 3: unstake
	ActionMintAndStake                 // 4: mint and stake
	ActionCreatePosition Action = 5    // >4: create position
)

type SwapInfo struct {
	Action          Action `json:"action"`
	UnwindingEpochs int    `json:"unwindingEpochs"`
}
