package angstrom

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type HookConfig struct {
	HTTP            HTTPConfig            `mapstructure:"http"`
	BlocksInFuture  int                   `mapstructure:"blocksInFuture"`
	RefreshInterval durationjson.Duration `mapstructure:"refreshInterval"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"baseUrl"`
	APIKey     string                `mapstructure:"apiKey"`
	RetryCount int                   `mapstructure:"retryCount"`
	Timeout    durationjson.Duration `mapstructure:"timeout"`
}
