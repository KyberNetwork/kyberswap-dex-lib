package uniswapv3

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	TickLensAddress    string `json:"tickLensAddress"`
	PreGenesisPoolPath string `json:"preGenesisPoolPath"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
	preGenesisPoolIDs  []string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
