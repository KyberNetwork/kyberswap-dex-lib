package hashflowv3

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type HTTPClientConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"baseUrl"`
	Source     string                `mapstructure:"source" json:"source"`
	APIKey     string                `mapstructure:"api_key" json:"apiKey"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retryCount"`
}
