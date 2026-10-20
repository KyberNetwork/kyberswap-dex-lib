package biconomyprop

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x0000009E5F98cDB4dd1E1d2E15B220217c88B3b8 on Base.
	Venue string `json:"venue"`

	// Feed is Biconomy's venue board feed. Unset, the tracker reads every board on-chain.
	Feed FeedConfig `json:"feed"`
}

// FeedConfig points at the venue board feed (wss://<api-host>/venue-stream).
type FeedConfig struct {
	URL string `json:"url"`
	// APIKey is sent only as the "x-api-key" handshake header. It stays in the tracker's config
	// and never enters pool state.
	APIKey string `json:"apiKey"`
	// MaxAgeMs is how old a snapshot may be and still be used (default defaultFeedMaxAge). An
	// older one, or none, sends the tracker back to reading the venue on-chain.
	MaxAgeMs int64 `json:"maxAgeMs"`
}

func (c *Config) feedMaxAge() time.Duration {
	if c.Feed.MaxAgeMs > 0 {
		return time.Duration(c.Feed.MaxAgeMs) * time.Millisecond
	}
	return defaultFeedMaxAge
}
