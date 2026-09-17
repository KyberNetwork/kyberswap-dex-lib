package lunyafun

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config wires the pool-factory decoder and the tracker. Historical discovery (start block, scan
// window) belongs to pool-service's backfill config driving poolfactory.FilterLogsBackfiller.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Factory is the LunyaLaunchFactory address whose LaunchCreated events list the launches.
	Factory string `json:"factory"`
}
