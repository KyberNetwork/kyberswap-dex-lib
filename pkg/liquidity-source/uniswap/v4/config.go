package uniswapv4

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	ChainID                valueobject.ChainID `json:"chainID"`
	DexID                  string              `json:"dexID"`
	SubgraphAPI            string              `json:"subgraphAPI"`
	UniversalRouterAddress string              `json:"universalRouterAddress"`
	Permit2Address         string              `json:"permit2Address"`
	Multicall3Address      string              `json:"multicall3Address"`
	StateViewAddress       string              `json:"stateViewAddress"`
	// PoolManagerAddress is the dex's own PoolManager; empty means the official PoolManager[chainID].
	PoolManagerAddress string `json:"poolManagerAddress"`
	// QuoterAddress is the fork's own V4 Quoter, bound to its PoolManager; empty means Quoter[chainID].
	QuoterAddress      string `json:"quoterAddress"`
	NewPoolLimit       int    `json:"newPoolLimit"`
	AllowSubgraphError bool   `json:"allowSubgraphError"`

	TimeThresholdByPool map[string]time.Duration `json:"timeThreshold"` // blocks swap after any event

	FetchTickFromStateView bool // instead of fetching from subgraph

	// UseLens reads pools through V4Lens: their price, liquidity and every
	// tick from one block in one eth_call, or a few when the ticks do not fit. A bootstrap then asks
	// no subgraph, and ticks accumulated from logs that turn out invalid are replaced by the lens'
	// set, which finds a tick the stored set lacked, rather than refetched index by index.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 leaves it to the node's eth_call cap.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard ephemeral.GuardConfig `json:"lensGuard,omitempty"`

	HookConfigs map[common.Address]any `json:"hookConfigs" mapstructure:"hookConfigs"`
}

func (c *Config) poolManager() *common.Address {
	if !common.IsHexAddress(c.PoolManagerAddress) {
		return nil
	}
	a := common.HexToAddress(c.PoolManagerAddress)
	return &a
}

func (c *Config) IsAllowSubgraphError() bool {
	return c.AllowSubgraphError
}
