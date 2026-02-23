package axima

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`

	// MaxAge is the maximum age of the pool data in seconds.
	// If the pool state is older than this (p.Timestamp + MaxAge < CurrentTime),
	// it will be considered stale and will not be used for trading.
	MaxAge int64 `json:"maxAge"`

	HTTPConfig HTTPConfig `json:"httpConfig"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"baseUrl" json:"baseUrl,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retryCount" json:"retryCount,omitempty"`
	APIKey     string                `mapstructure:"apiKey" json:"apiKey"`
}
