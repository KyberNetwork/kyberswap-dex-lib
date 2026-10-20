package upshiftusdf

import "github.com/holiman/uint256"

// Extra is the vault state deposit() reads. Apr and TokenPrice are the _records[CurrentPeriod]
// entry; compute() copies Apr forward and grows the price by a fixed step per rolled day.
// The vault has no pause flag. MaxSupply 0 means unlimited (canMint).
type Extra struct {
	StartOfYear   uint64       `json:"s"`
	CurrentPeriod uint64       `json:"cp"`
	Apr           *uint256.Int `json:"apr"`
	TokenPrice    *uint256.Int `json:"p"`
	MinDeposit    *uint256.Int `json:"min"`
	MaxSupply     *uint256.Int `json:"max"`
	TotalSupply   *uint256.Int `json:"ts"`
}

// SwapInfo is what deposit() changes: the rolled days, the rolled price and the minted USDF.
type SwapInfo struct {
	Days     uint64       `json:"-"`
	NewPrice *uint256.Int `json:"-"`
	Shares   *uint256.Int `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
