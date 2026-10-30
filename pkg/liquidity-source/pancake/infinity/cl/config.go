package cl

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"

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

	FetchTickFromRPC bool // instead of fetching from subgraph

	// UseLens reads pools through PancakeInfinityCLLens: their price, liquidity,
	// tick spacing and every tick from one block in one eth_call, or a few when the ticks do not
	// fit. A bootstrap then asks no subgraph and needs no logs, and ticks accumulated from logs that
	// turn out invalid are replaced by the lens's set, which finds a tick the stored set lacked,
	// rather than refetched index by index.
	ephemeral.LensConfig

	StableHookFactories []string `json:"stableHookFactories,omitempty"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
