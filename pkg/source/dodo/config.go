package dodo

type Config struct {
	DexID             string `json:"dexID"`
	SubgraphAPI       string `json:"subgraphAPI"`
	NewPoolLimit      int    `json:"newPoolLimit"`
	DodoV1SellHelper  string `json:"dodoV1SellHelper"`
}
