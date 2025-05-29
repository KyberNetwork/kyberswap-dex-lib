package nativev1

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/go-resty/resty/v2"
)

type HTTPClientConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count"`
	APIKey     string                `mapstructure:"api_key" json:"api_key"`
	Client     *resty.Client
}
