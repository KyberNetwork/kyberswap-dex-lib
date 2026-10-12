package shared

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID           string              `json:"dexID,omitempty"`
	ChainID         valueobject.ChainID `json:"chainID,omitempty"`
	PoolType        string              `json:"poolType,omitempty"`
	SubgraphAPI     string              `json:"subgraphAPI,omitempty"`
	SubgraphHeaders http.Header         `json:"subgraphHeaders,omitempty"`
	NewPoolLimit    int                 `json:"newPoolLimit,omitempty"`
	VaultExplorer   string              `json:"vaultExplorer"`
	SubgraphChain   string              `json:"subgraphChain"`
	// Factories lists pool factories for on-chain discovery, used instead of the subgraph when set.
	Factories        []string `json:"factories,omitempty"`
	SubgraphPoolType string   `json:"-"`
}
