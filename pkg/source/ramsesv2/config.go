package ramsesv2

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
