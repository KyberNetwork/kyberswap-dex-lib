package v3

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders,omitempty"`
	AllowSubgraphError bool        `json:"allowSubgraphError,omitempty"`
	TickLensAddress    string      `json:"tickLensAddress,omitempty"`
	AlwaysUseTickLens  bool        `json:"alwaysUseTickLens,omitempty"` // instead of fetching from subgraph
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
