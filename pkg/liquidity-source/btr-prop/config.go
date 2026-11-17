package btrprop

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is the dex-lib exchange configuration.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	Factory string              `json:"factory"`
	// Executor is the aggregator executor that calls swap_qe; empty means DefaultExecutor. Its
	// membership in the pool's swap gate decides whether a SWAP_GATED leg is open to us.
	Executor string `json:"executor"`
}

// NewConfig returns the Monad defaults.
func NewConfig(chainID valueobject.ChainID) *Config {
	return &Config{DexID: DexType, ChainID: chainID, Factory: PoolFactory, Executor: DefaultExecutor}
}

// executor is the configured swap_qe caller, DefaultExecutor when unset.
func (c *Config) executor() common.Address {
	if c.Executor == "" {
		return common.HexToAddress(DefaultExecutor)
	}
	return common.HexToAddress(c.Executor)
}
