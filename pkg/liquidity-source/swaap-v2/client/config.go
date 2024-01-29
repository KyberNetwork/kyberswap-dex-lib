package client

import "time"

type HTTPClientConfig struct {
	BaseURL    string        `mapstructure:"baseUrl" json:"baseUrl"`
	Timeout    time.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int           `mapstructure:"retryCount" json:"retryCount"`
}
