package axima

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID   string `json:"dexID"`
	Chain   string `json:"chain"`
	ChainID int64  `json:"chainID"`

	// MaxAge is the maximum age of the pool data in seconds.
	// If the pool state is older than this (p.Timestamp + MaxAge < CurrentTime),
	// it will be considered stale and will not be used for trading.
	MaxAge int64 `json:"maxAge"`

	// FeedMaxAge is the maximum age, in seconds, of the on-chain price feed that
	// the pool's quoter extension reads during a swap. When the feed is older than
	// this the pool is treated as unswappable, because the extension reverts
	// FeedStalled() and takes the whole swap down with it. Zero disables the check.
	FeedMaxAge int64 `json:"feedMaxAge"`

	IsV2 bool `json:"isV2"`

	HTTPConfig HTTPConfig `json:"httpConfig"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"baseUrl" json:"baseUrl,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retryCount" json:"retryCount,omitempty"`
	APIKey     string                `mapstructure:"apiKey" json:"apiKey"`
}
