package flowstatec1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// PoolConfig seeds one inventory token. FlowState has no subgraph on Robinhood yet
// and staging depth is a single pool, so the pool list is config-seeded (real pool
// address is resolved on-chain via poolByToken, not hardcoded here).
type PoolConfig struct {
	Token       string `json:"token"`
	ProbeAmount string `json:"probeAmount"`
}

type Config struct {
	DexID       string              `json:"dexID"`
	ChainID     valueobject.ChainID `json:"chainID"`
	Market      string              `json:"market"`
	QuoteAssets []string            `json:"quoteAssets"`
	Pools       []PoolConfig        `json:"pools"`
}
