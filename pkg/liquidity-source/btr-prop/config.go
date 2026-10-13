package btrprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config is the dex-lib exchange configuration.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	Factory string              `json:"factory"`
	// Executor is the aggregator executor that calls swap_qe. Its registry lane
	// decides whether a SWAP_GATED pool is open to us.
	Executor string `json:"executor"`
}

// NewConfig returns the Monad defaults.
func NewConfig(chainID valueobject.ChainID) *Config {
	return &Config{DexID: DexType, ChainID: chainID, Factory: PoolFactory}
}
