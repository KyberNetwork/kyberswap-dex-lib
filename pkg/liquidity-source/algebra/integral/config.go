package integral

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	AlwaysUseTickLens bool
	TickLensAddress   string

	UseBasePluginV2 bool `json:"useBasePluginV2"`

	// UseLens reads pools through AlgebraIntegralLens: their price, liquidity and
	// every tick from one block, walking the pool's tick list. A bootstrap then asks no subgraph, a
	// pool created without state gets its whole state, and ticks gathered from logs that turn out
	// invalid are replaced by the pool's own set, which re-reading the stored indexes cannot do.
	// Its plugin is read through AlgebraIntegralPluginLens: the settings and the
	// oracle's timepoints in one call, where multicalls take one for the settings and one for each
	// page of timepoints, falling back to them on any lens error.
	ephemeral.LensConfig
}
