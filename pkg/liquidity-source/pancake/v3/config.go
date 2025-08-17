package pancakev3

import (
	"net/http"

	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	TrackInactivePools *pooltrack.TrackInactivePoolsConfig `json:"trackInactivePools"`

	AlwaysUseTickLens bool // instead of fetching from subgraph
	TickLensAddress   string
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
