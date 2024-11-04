package uniswapv3

import "net/http"

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	TickLensAddress    string      `json:"tickLensAddress"`
	PreGenesisPoolPath string      `json:"preGenesisPoolPath"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
	preGenesisPoolIDs  []string

	AlwaysUseTickLens bool // instead of fetching from subgraph
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
