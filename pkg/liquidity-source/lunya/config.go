package lunya

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config wires the pool-factory decoder and the tracker. Historical discovery (start block, scan
// window) belongs to pool-service's backfill config driving poolfactory.FilterLogsBackfiller.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Factory is the LunyaPoolFactory address whose PoolCreated events list every pool, of every type.
	Factory string `json:"factory"`
}
