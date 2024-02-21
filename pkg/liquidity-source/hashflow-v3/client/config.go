package client

import "time"

type HTTPClientConfig struct {
	BaseURL    string        `mapstructure:"base_url" json:"baseUrl"`
	Source     string        `mapstructure:"source" json:"source"`
	APIKey     string        `mapstructure:"api_key" json:"apiKey"`
	Timeout    time.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int           `mapstructure:"retry_count" json:"retryCount"`
}
