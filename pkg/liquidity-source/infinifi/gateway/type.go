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
	ActionMint             Action = iota // 0: mint => USDC → iUSD
	ActionRedeem                         // 1: redeem => iUSD → USDC
	ActionStake                          // 2: stake => iUSD → siUSD
	ActionUnstake                        // 3: unstake => siUSD → iUSD
	ActionMintAndStake                   // 4: mint and stake => USDC → siUSD
	ActionCreatePosition                 // 5: create position => iUSD → liUSD
	ActionMintAndLock                    // 6: mint and lock => USDC → liUSD
	ActionUnstakeAndRedeem               // 7: unstake and redeem => siUSD → USDC
)

const (
	// https://github.com/InfiniFi-Labs/infinifi-protocol/blob/master/deployment/configuration/addresses.1.json
	MintControllerAddress   = "0x49877d937B9a00d50557bdC3D87287b5c3a4C256"
	RedeemControllerAddress = "0xCb1747E89a43DEdcF4A2b831a0D94859EFeC7601"
	LockControllerAddress   = "0x1d95cC100D6Cd9C7BbDbD7Cb328d99b3D6037fF7"
)

type SwapInfo struct {
	Action            Action   `json:"action"`
	QuoterControllers []string `json:"quoterControllers"`
	UnwindingEpochs   int      `json:"unwindingEpochs"`
}
