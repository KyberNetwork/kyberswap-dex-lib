package ponsv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is this package's pool-list/pool-tracker wiring config.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Factory is the launchpad-deployment PonsV2LaunchFactory address. The
	// migration deployment is out of scope for now: as of the discovery
	// pass this integration was built from, its factory has emitted zero
	// TokenLaunched events, so there is nothing to discover there yet.
	Factory string `json:"factory"`
	// NewPoolLimit caps how many TokenLaunched logs are processed per
	// GetNewPools call.
	NewPoolLimit int `json:"newPoolLimit"`
}
