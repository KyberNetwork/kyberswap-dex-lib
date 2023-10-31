package liquiditybookv20

type Config struct {
	DexID              string `json:"dexID"`
	FactoryAddress     string `json:"factoryAddress"`
	NewPoolLimit       int    `json:"newPoolLimit"`
	SubgraphAPI        string `json:"subgraphAPI"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
}
