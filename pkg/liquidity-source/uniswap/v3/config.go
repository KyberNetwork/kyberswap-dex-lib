package uniswapv3

import (
	"net/http"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	ChainID            valueobject.ChainID `json:"chainId"`
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders,omitempty"`
	AllowSubgraphError bool        `json:"allowSubgraphError,omitempty"`
	TickLensAddress    string      `json:"tickLensAddress,omitempty"`
	PreGenesisPoolPath string      `json:"preGenesisPoolPath,omitempty"`
	AlwaysUseTickLens  bool        `json:"alwaysUseTickLens,omitempty"` // instead of fetching from subgraph

	// UseLens reads pools through V3Lens: their state and every tick from one
	// block in one eth_call, or a few when the ticks do not fit. A bootstrap then asks no subgraph,
	// and ticks accumulated from logs that turn out invalid are replaced by the lens's set, which
	// finds a tick the stored set lacked, rather than refetched index by index.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 leaves it to the node's eth_call cap.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard"`

	// SubgraphPoolField overrides the subgraph's top-level pools entity name
	// (e.g. ramses-style deployments querying something other than `pools`).
	// Empty means the default `pools`.
	SubgraphPoolField string `json:"subgraphPool,omitempty"`

	ForksConfig map[string]ForkConfig `json:"forksConfig,omitempty"`

	preGenesisPoolIDs []string
}

type ForkConfig struct {
	// pons-fun
	Multicall3 string `json:"multicall3"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
