package weighted

import "net/http"

type Config struct {
	DexID                 string      `json:"dexID"`
	ProtocolFeesCollector string      `json:"protocolFeesCollector"`
	SubgraphAPI           string      `json:"subgraphAPI"`
	SubgraphHeaders       http.Header `json:"subgraphHeaders"`
	NewPoolLimit          int         `json:"newPoolLimit"`
}
