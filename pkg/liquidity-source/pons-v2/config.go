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
	// StartBlock is the first block GetNewPools scans from on a cold start
	// (empty metadata) -- normally the Factory contract's deployment block,
	// so discovery never has to walk pre-deployment blocks that can't
	// contain a TokenLaunched log.
	StartBlock uint64 `json:"startBlock"`
	// MaxBlockRangePerScan caps a single eth_getLogs call's block span, and
	// so bounds how many TokenLaunched logs (and how large a single
	// buildPools multicall batch) one GetNewPools call can produce. Public
	// RPC providers commonly cap getLogs windows (often 1-10k blocks);
	// operators should tune this to both the RPC's actual limit and the
	// expected launch cadence on the target chain. All launches found
	// within the window are always processed -- there is no separate
	// per-call count limit, since capping by count independently of the
	// scanned block range risks a single over-limit block stalling
	// discovery forever (the same over-limit set would be re-fetched and
	// re-truncated every pass). Defaults to defaultMaxBlockRangePerScan
	// when unset.
	MaxBlockRangePerScan uint64 `json:"maxBlockRangePerScan"`
}
