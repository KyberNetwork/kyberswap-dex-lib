package algebrav1

type Config struct {
	DexID              string
	SubgraphAPI        string `json:"subgraphAPI"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`
}
