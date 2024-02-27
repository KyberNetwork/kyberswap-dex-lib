package bancor_v21

type ExtraInner struct {
	anchorAddress string `json:"anchorAddress"`
	conversionFee uint64 `json:"conversionFee"`
}

type Extra struct {
	AnchorMap                 map[string]struct{} `json:"anchorMap"`
	AnchorsByConvertibleToken map[string][]string `json:"anchorsByConvertibleToken"`
}

type Gas struct {
	Swap int64
}

type PoolMetaInner struct {
	Fee         uint64 `json:"fee"`
	BlockNumber uint64 `json:"blockNumber"`
}
