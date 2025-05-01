package shared

import "net/http"

type Config struct {
	DexID                 string      `json:"dexID"`
	SubgraphChain         string      `json:"subgraphChain"`
	SubgraphPoolTypes     []string    `json:"-"`
	SubgraphAPI           string      `json:"subgraphAPI"`
	SubgraphHeaders       http.Header `json:"subgraphHeaders"`
	NewPoolLimit          int         `json:"newPoolLimit"`
	BatchSwapEnabled      bool        `json:"batchSwapEnabled"`
	ProtocolFeesCollector string      `json:"protocolFeesCollector"`
	UseSubgraphV1         bool        `json:"useSubgraphV1"`
}
