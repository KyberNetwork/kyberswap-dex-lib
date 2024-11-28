package integral

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
	SkipFeeCalculating bool        `json:"skipFeeCalculating"` // do not pre-calculate fee at tracker, use last block's fee instead
	UseDirectionalFee  bool        `json:"useDirectionalFee"`  // for Camelot and similar dexes

	AlwaysUseTickLens bool
	TickLensAddress   string

	UseBasePluginV2 bool
}
