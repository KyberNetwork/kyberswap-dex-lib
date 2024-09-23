package shared

import "net/http"

type Config struct {
	DexID             string      `json:"dexID"`
	SubgraphAPI       string      `json:"subgraphAPI"`
	SubgraphHeaders   http.Header `json:"subgraphHeaders"`
	NewPoolLimit      int         `json:"newPoolLimit"`
	DodoV1SellHelper  string      `json:"dodoV1SellHelper"`
	BlacklistFilePath string      `json:"blacklistFilePath"`
}
