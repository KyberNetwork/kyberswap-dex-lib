package v3

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

type Config struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders,omitempty"`
	AllowSubgraphError bool        `json:"allowSubgraphError,omitempty"`
	TickLensAddress    string      `json:"tickLensAddress,omitempty"`
	AlwaysUseTickLens  bool        `json:"alwaysUseTickLens,omitempty"` // instead of fetching from subgraph
	ExecutorAddress    string      `json:"executorAddress,omitempty"`

	// UseLens reads pools through V3Lens: their state and every tick from one
	// block in one eth_call, or a few when the ticks do not fit. A bootstrap then asks no subgraph,
	// and ticks from logs that turn out invalid are replaced by the lens's set, which finds a tick
	// the stored set lacked, rather than refetched index by index.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 leaves it to the node's eth_call cap.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard,omitempty"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
