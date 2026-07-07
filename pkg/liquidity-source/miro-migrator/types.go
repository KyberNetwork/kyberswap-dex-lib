package miromigrator

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	Paused bool `json:"p"`
}

type SwapInfo struct {
	IsDeposit bool `json:"isDeposit"`
}
