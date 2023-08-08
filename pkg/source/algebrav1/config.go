package algebrav1

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
	SkipFeeCalculating bool   `json:"skipFeeCalculating"` // do not pre-calculate fee at tracker, use last block's fee instead
	UseDirectionalFee  bool   `json:"useDirectionalFee"`  // for Camelot and similar dexes
}
