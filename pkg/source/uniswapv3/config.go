package uniswapv3

import (
	"net/http"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID              string
	SubgraphAPI        string                    `json:"subgraphAPI,omitempty"`
	SubgraphHeaders    http.Header               `json:"subgraphHeaders,omitempty"`
	AllowSubgraphError bool                      `json:"allowSubgraphError,omitempty"`
	TickLensAddress    string                    `json:"tickLensAddress,omitempty"`
	PreGenesisPoolPath string                    `json:"preGenesisPoolPath,omitempty"`
	AlwaysUseTickLens  bool                      `json:"alwaysUseTickLens,omitempty"` // instead of fetching from subgraph
	TrackInactivePools *TrackInactivePoolsConfig `json:"trackInactivePools,omitempty"`

	preGenesisPoolIDs []string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}

type TrackInactivePoolsConfig struct {
	Enabled       bool                  `json:"enabled"`
	TimeThreshold durationjson.Duration `json:"timeThreshold"`
}
