package algebrav1

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
	SkipFeeCalculating bool        `json:"skipFeeCalculating"` // do not pre-calculate fee at tracker, use last block's fee instead
	UseDirectionalFee  bool        `json:"useDirectionalFee"`  // for Camelot and similar dexes

	AlwaysUseTickLens bool
	TickLensAddress   string

	// UseLens reads pools through AlgebraV1Lens: their state and every tick from
	// one block in one eth_call, or a few when the ticks do not fit. A bootstrap then asks no
	// subgraph, a pool without state gets its state on its first update, and ticks accumulated from
	// logs that turn out invalid are replaced by the lens's set, which finds a tick the stored set
	// lacked, rather than refetched index by index.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 takes the node's eth_call cap, up to
	// ephemeral.CallGasCeiling.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard"`
}
