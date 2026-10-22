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
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 takes the node's eth_call cap, up to
	// ephemeral.CallGasCeiling.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard"`
	// LensBatch gathers the first lens read of pools read at the same block into one call, with
	// every exchange that reads through the same node; nil reads each pool alone.
	LensBatch *ephemeral.BatchConfig `json:"lensBatch,omitempty"`
}
