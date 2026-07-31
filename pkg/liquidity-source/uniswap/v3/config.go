package uniswapv3

import (
	"net/http"

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
