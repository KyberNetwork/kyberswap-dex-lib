package angstrom

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type RFQConfig struct {
	HTTP           HTTPConfig            `mapstructure:"http"`
	BlocksInFuture int                   `mapstructure:"blocksInFuture"`
	CacheTTL       durationjson.Duration `mapstructure:"cacheTtl"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"baseUrl"`
	APIKey     string                `mapstructure:"apiKey"`
	RetryCount int                   `mapstructure:"retryCount"`
	Timeout    durationjson.Duration `mapstructure:"timeout"`
}
