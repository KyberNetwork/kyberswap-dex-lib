package ramsesv2

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	AlwaysUseTickLens bool // instead of fetching from subgraph
	TickLensAddress   string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
