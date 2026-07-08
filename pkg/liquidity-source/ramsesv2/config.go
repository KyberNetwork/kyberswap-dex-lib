package ramsesv2

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	SubgraphPool       string      `json:"subgraphPool"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	AlwaysUseTickLens bool // instead of fetching from subgraph
	TickLensAddress   string
	IsPoolV3          bool `json:"isPoolV3"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
