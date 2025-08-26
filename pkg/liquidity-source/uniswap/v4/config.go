package uniswapv4

import (
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
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

	TimeThresholdByPool map[string]time.Duration  `json:"timeThreshold"` // blocks swap after any event
	TrackInactivePools  *TrackInactivePoolsConfig `json:"trackInactivePools"`

	FetchTickFromStateView bool // instead of fetching from subgraph
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}

type TrackInactivePoolsConfig struct {
	Enabled               bool                  `json:"enabled"`
	TimeThreshold         durationjson.Duration `json:"timeThreshold"`
	ZoraHookTimeThreshold durationjson.Duration `json:"zoraHookTimeThreshold"`
}
