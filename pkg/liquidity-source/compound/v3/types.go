package v3

type Extra struct {
	IsWithdrawPaused bool `json:"isWithdrawPaused"`
	IsSupplyPaused   bool `json:"isSupplyPaused"`
}

type PoolMeta struct {
	BlockNumber uint64
}
