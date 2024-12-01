package integral

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	// AlwaysUseTickLens bool
	// TickLensAddress   string

	UseBasePluginV2 bool
}
