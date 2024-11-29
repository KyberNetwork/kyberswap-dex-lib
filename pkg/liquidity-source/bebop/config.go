package bebop

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL       string                `mapstructure:"base_url" json:"base_url"`
	Timeout       durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	Debug         bool                  `mapstructure:"debug" json:"debug"`
	RetryCount    int                   `mapstructure:"retry_count" json:"retry_count"`
	Name          string                `mapstructure:"name" json:"name"`
	Authorization string                `mapstructure:"authorization" json:"authorization"`
}
