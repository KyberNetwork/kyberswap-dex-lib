package ponsv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is this package's pool-factory/pool-tracker wiring config.
// Historical discovery (FilterLogs block-range scanning, start block, max
// scan window) is owned by pool-service's dependencies:/Backfill config and
// poolfactory.FilterLogsBackfiller, not by this package -- see pool_factory.go.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Factory is the launchpad-deployment PonsV2LaunchFactory address. The
	// migration deployment is out of scope for now: as of the discovery
	// pass this integration was built from, its factory has emitted zero
	// TokenLaunched events, so there is nothing to discover there yet.
	Factory string `json:"factory"`
}
