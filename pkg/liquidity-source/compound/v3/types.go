package v3

type Extra struct {
	IsWithdrawPaused bool `json:"isWithdrawPaused,omitempty"`
	IsSupplyPaused   bool `json:"isSupplyPaused,omitempty"`
}

type PoolMeta struct {
	BlockNumber uint64
}
