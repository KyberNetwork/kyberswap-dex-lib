package upshiftvaultv2

import "github.com/holiman/uint256"

// Extra is the state deposit() checks. Valuation is whitelist.getTotalAssetsValuation(externalAssets)
// called by the vault: the denominator of previewDeposit. TotalAssets is vault.getTotalAssets(), the
// total compared with DepositCap; it equals Valuation on HLPe and is net of collectable fees on earnAUSD.
type Extra struct {
	DepositsPaused bool         `json:"p,omitempty"`
	SenderGated    bool         `json:"g,omitempty"`
	Supply         *uint256.Int `json:"s"`
	Valuation      *uint256.Int `json:"v"`
	TotalAssets    *uint256.Int `json:"ta"`
	MaxDeposit     *uint256.Int `json:"md"`
	DepositCap     *uint256.Int `json:"dc"`
}

type SwapInfo struct {
	Shares *uint256.Int `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
