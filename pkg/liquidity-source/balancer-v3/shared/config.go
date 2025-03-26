package shared

import (
	"net/http"
)

type Config struct {
	DexID           string      `json:"dexID,omitempty"`
	PoolType        string      `json:"poolType,omitempty"`
	SubgraphAPI     string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders http.Header `json:"subgraphHeaders,omitempty"`
	NewPoolLimit    int         `json:"newPoolLimit,omitempty"`
	VaultExplorer   string      `json:"vaultExplorer"`
	Factory         string      `json:"factory,omitempty"`
	DefaultHook     string      `json:"defaultHook,omitempty"`
}
