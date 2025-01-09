package bebop

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL       string                `mapstructure:"base_url" json:"baseUrl"`
	Timeout       durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount    int                   `mapstructure:"retry_count" json:"retryCount"`
	Name          string                `mapstructure:"name" json:"name"`
	Authorization string                `mapstructure:"authorization" json:"authorization"`
}
