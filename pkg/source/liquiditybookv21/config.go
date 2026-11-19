package liquiditybookv21

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

type Config struct {
	DexID              string      `json:"dexID"`
	FactoryAddress     string      `json:"factoryAddress"`
	NewPoolLimit       int         `json:"newPoolLimit"`
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`

	// UseLens reads pairs through LiquidityBookV21Lens: the active bin, the
	// reserves and every non-empty bin from one block in one eth_call, or a few when the bins do
	// not fit. A bootstrap then asks no subgraph, and bins from logs that no longer add up to the
	// reserves are replaced by the lens's set, which finds a bin the stored set lacked, rather than
	// read again id by id.
	ephemeral.LensConfig
}
