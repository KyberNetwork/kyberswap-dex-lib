package stable

import "net/http"

type Config struct {
	DexID           string      `json:"dexID"`
	SubgraphAPI     string      `json:"subgraphAPI"`
	SubgraphHeaders http.Header `json:"subgraphHeaders"`
	NewPoolLimit    int         `json:"newPoolLimit"`
	VaultExplorer   string      `json:"vaultExplorer"`
	Factory         string      `json:"factory"`
	DefaultHook     string      `json:"defaultHook"`
}
