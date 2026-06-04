package bouncetech

import "github.com/holiman/uint256"

// Extra holds the per-block dynamic state fetched by the pool tracker.
type Extra struct {
	ExchangeRate       *uint256.Int `json:"exchangeRate"`
	RedemptionFee      *uint256.Int `json:"redemptionFee"`
	TargetLeverage     *uint256.Int `json:"targetLeverage"`
	MinTransactionSize *uint256.Int `json:"minTransactionSize"`
	MintPaused         bool         `json:"mintPaused"`
}

// StaticExtra holds immutable per-pool data set once at discovery time.
type StaticExtra struct {
	USDC string `json:"usdc"`
}

// SwapInfo carries the per-swap data attached to CalcAmountOutResult.SwapInfo.
// The aggregator router reads this to build the executor calldata.
type SwapInfo struct {
	IsMint bool `json:"isMint"`
}
