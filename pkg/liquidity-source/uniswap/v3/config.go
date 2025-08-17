package uniswapv3

import (
	"net/http"

	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders,omitempty"`
	AllowSubgraphError bool        `json:"allowSubgraphError,omitempty"`
	TickLensAddress    string      `json:"tickLensAddress,omitempty"`
	PreGenesisPoolPath string      `json:"preGenesisPoolPath,omitempty"`
	AlwaysUseTickLens  bool        `json:"alwaysUseTickLens,omitempty"` // instead of fetching from subgraph

	TrackInactivePools *pooltrack.TrackInactivePoolsConfig `json:"trackInactivePools"`

	preGenesisPoolIDs []string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
