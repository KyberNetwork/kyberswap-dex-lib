package bin

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"

type Config struct {
	ChainID                int    `json:"chainID"`
	DexID                  string `json:"dexID"`
	SubgraphAPI            string `json:"subgraphAPI"`
	UniversalRouterAddress string `json:"universalRouterAddress"`
	Permit2Address         string `json:"permit2Address"`
	Multicall3Address      string `json:"multicall3Address"`
	VaultAddress           string `json:"vaultAddress"`
	BinPoolManagerAddress  string `json:"binPoolManagerAddress"`
	NewPoolLimit           int    `json:"newPoolLimit"`
	AllowSubgraphError     bool   `json:"allowSubgraphError"`

	// UseLens reads pools through PancakeInfinityBinLens: the active bin and
	// every non-empty bin from one block in one eth_call, or a few when the bins do not fit. A
	// bootstrap then asks no subgraph, a pool without bins gets them on its first update, and a
	// stored set found to lack a bin is replaced by the lens's set, which finds a bin the stored set
	// never had, rather than read again id by id.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 leaves it to the node's eth_call cap.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard,omitempty"`
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
