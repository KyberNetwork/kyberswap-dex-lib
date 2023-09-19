package uniswapv3pt

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	PoolTicksAPI       string `json:"poolTicksAPI"`
	TickLensAddress    string `json:"tickLensAddress"`
	PreGenesisPoolPath string `json:"preGenesisPoolPath"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
	preGenesisPoolIDs  []string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
