package slyngfun

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config wires the pool-factory decoder and the tracker. Historical discovery (start block, scan
// window) belongs to pool-service's backfill config driving poolfactory.FilterLogsBackfiller.
//
// Robinhood Chain's public RPC silently truncates eth_getLogs results above roughly one million
// blocks instead of erroring, so MaxBlockRangePerScan for this source must stay well under that;
// Slyng's own indexer scans 200,000 blocks at a time.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Launchpad is the Slyng Launchpad address. It is the factory that emits TokenCreated, the
	// contract that holds every curve, and the one that buy() and sell() are called on.
	Launchpad string `json:"launchpad"`
}
