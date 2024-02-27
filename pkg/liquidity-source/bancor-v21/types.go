package bancor_v21

type Extra struct {
	anchorAddress string `json:"anchorAddress"`
	conversionFee uint64 `json:"conversionFee"`
}

type Gas struct {
	Swap int64
}

type PoolMeta struct {
	Fee         uint64 `json:"fee"`
	BlockNumber uint64 `json:"blockNumber"`
}
