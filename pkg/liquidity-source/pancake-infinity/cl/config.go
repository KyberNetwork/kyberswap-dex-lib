package cl

type Config struct {
	ChainID                int    `json:"chainID"`
	DexID                  string `json:"dexID"`
	SubgraphAPI            string `json:"subgraphAPI"`
	UniversalRouterAddress string `json:"universalRouterAddress"`
	Permit2Address         string `json:"permit2Address"`
	Multicall3Address      string `json:"multicall3Address"`
	CLPoolManagerAddress   string `json:"clPoolManagerAddress"`
	NewPoolLimit           int    `json:"newPoolLimit"`
	AllowSubgraphError     bool   `json:"allowSubgraphError"`

	FetchTickFromRPC bool // instead of fetching from subgraph
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
