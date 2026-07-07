package beets_ss

import "math/big"

type Extra struct {
	TotalSupply   *big.Int `json:"total_supply"`
	TotalAssets   *big.Int `json:"total_asset"`
	DepositPaused bool     `json:"deposit_paused"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Gas struct {
	Swap int64
}
