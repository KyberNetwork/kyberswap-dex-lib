package poolsidev1

type Gas struct {
	Swap int64
}

type Extra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}
