package cl

type Config struct {
	ChainID                int    `json:"chainID"`
	DexID                  string `json:"dexID"`
	SubgraphAPI            string `json:"subgraphAPI"`
	UniversalRouterAddress string `json:"universalRouterAddress"`
	Permit2Address         string `json:"permit2Address"`
	Multicall3Address      string `json:"multicall3Address"`
	VaultAddress           string `json:"vaultAddress"`
	CLPoolManagerAddress   string `json:"clPoolManagerAddress"`
	NewPoolLimit           int    `json:"newPoolLimit"`
	AllowSubgraphError     bool   `json:"allowSubgraphError"`

	// Used when SubgraphAPI is empty: pools come from the manager's Initialize logs.
	StartBlock    uint64 `json:"startBlock"`
	LogBlockRange uint64 `json:"logBlockRange"`

	FetchTickFromRPC bool // instead of fetching from subgraph

	StableHookFactories []string `json:"stableHookFactories,omitempty"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
