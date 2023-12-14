package ramsesv2

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	TickLensAddress    string `json:"tickLensAddress"`
	PreGenesisPoolPath string `json:"preGenesisPoolPath"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
