package nemoprop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config is the per-dex-id configuration, seen only by the lister and the
// tracker. The live feed runs in the tracker's process only; simulators are
// built from the ladders and timing the tracker writes into pool state.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Address is the NemoSwap ERC1967 proxy: market discovery, swap
	// entrypoint and approval address.
	Address string `json:"address"`

	// Buffer scales every feed ladder output by Buffer/10000 (e.g. 9980
	// keeps 99.8%), to absorb drift between quote and execution. Values
	// outside 1-9999 disable it: a buffer never inflates quotes.
	Buffer int64 `json:"buffer"`

	// DecayBps is how much quotes worsen per second once a snapshot is past
	// Feed.FreshMs: outputs scale by 1/(1 + DecayBps/10000 * secondsStale).
	// 0 disables decay: quoting stops once the snapshot is no longer fresh.
	DecayBps int64 `json:"decayBps"`

	Feed FeedConfig `json:"feed"`
}

// FeedConfig points at Nemo's live quote websocket, the only price source.
type FeedConfig struct {
	URL string `json:"url"`
	// AuthToken is a read-only integrator key, sent only as an
	// "Authorization: Bearer" handshake header. It stays in the tracker's
	// config and never enters pool state.
	AuthToken string `json:"authToken"`
	// FreshMs is how long a snapshot quotes undecayed (default
	// defaultFeedFresh). Decay (see Config.DecayBps) starts after it.
	FreshMs int64 `json:"freshMs"`
	// MaxAgeMs is how long after receipt a snapshot quotes at all, decayed
	// or not (default defaultFeedMaxAge, or FreshMs if that's longer).
	MaxAgeMs int64 `json:"maxAgeMs"`
}

func (c *Config) freshMs() int64 {
	if c.Feed.FreshMs > 0 {
		return c.Feed.FreshMs
	}
	return defaultFeedFresh.Milliseconds()
}

// maxAgeMs bounds how long after receipt a snapshot quotes at all. Unset, it
// is defaultFeedMaxAge, or the fresh window if that's longer.
func (c *Config) maxAgeMs() int64 {
	if c.Feed.MaxAgeMs > 0 {
		return c.Feed.MaxAgeMs
	}
	return max(defaultFeedMaxAge.Milliseconds(), c.freshMs())
}
