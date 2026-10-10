package btr

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config is the dex-lib exchange configuration.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	Factory string              `json:"factory"`
}

// NewConfig returns the Monad defaults.
func NewConfig(chainID valueobject.ChainID) *Config {
	return &Config{DexID: DexType, ChainID: chainID, Factory: PoolFactory}
}
