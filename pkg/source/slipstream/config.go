package slipstream

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
