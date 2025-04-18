package stable

import "net/http"

type Config struct {
	DexID            string      `json:"dexID"`
	SubgraphAPI      string      `json:"subgraphAPI"`
	SubgraphHeaders  http.Header `json:"subgraphHeaders"`
	NewPoolLimit     int         `json:"newPoolLimit"`
	BatchSwapEnabled bool        `json:"batchSwapEnabled"`
}
