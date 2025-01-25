package uniswapv4

type Config struct {
	DexID              string `json:"dexID"`
	SubgraphAPI        string `json:"subgraphAPI"`
	PoolManagerAddress string `json:"poolManagerAddress"`
	NewPoolLimit       int    `json:"newPoolLimit"`
}
