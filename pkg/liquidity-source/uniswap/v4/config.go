package uniswapv4

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	ChainID                int    `json:"chainID"`
	DexID                  string `json:"dexID"`
	SubgraphAPI            string `json:"subgraphAPI"`
	UniversalRouterAddress string `json:"universalRouterAddress"`
	Permit2Address         string `json:"permit2Address"`
	Multicall3Address      string `json:"multicall3Address"`
	StateViewAddress       string `json:"stateViewAddress"`
	NewPoolLimit           int    `json:"newPoolLimit"`
	AllowSubgraphError     bool   `json:"allowSubgraphError"`

	TimeThresholdByPool map[string]time.Duration `json:"timeThreshold"` // blocks swap after any event

	FetchTickFromStateView bool // instead of fetching from subgraph

	HookConfigs map[common.Address]any `json:"hookConfigs" mapstructure:"hookConfigs"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
