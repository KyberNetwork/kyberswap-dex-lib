package axima

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type Config struct {
	DexID string `json:"dexID"`

	HTTPConfig HTTPConfig `json:"httpConfig"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"baseUrl" json:"baseUrl,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retryCount" json:"retryCount,omitempty"`
	APIKey     string                `mapstructure:"apiKey" json:"apiKey"`
}
