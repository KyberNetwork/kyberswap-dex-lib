package liquiditybookv20

type Config struct {
	DexID              string `json:"dexID"`
	FactoryAddress     string `json:"factoryAddress"`
	RouterAddress      string `json:"routerAddress"`
	NewPoolLimit       int    `json:"newPoolLimit"`
	SubgraphAPI        string `json:"subgraphAPI"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
}
