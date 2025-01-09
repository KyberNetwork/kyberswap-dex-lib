package bebop

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL       string                `mapstructure:"base_url" json:"base_url"`
	Timeout       durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount    int                   `mapstructure:"retry_count" json:"retry_count"`
	Name          string                `mapstructure:"name" json:"name"`
	Authorization string                `mapstructure:"authorization" json:"authorization"`
}

type Config struct {
	APIName string           `json:"apiName"`
	DexID   string           `json:"dexId"`
	HTTP    HTTPClientConfig `mapstructure:"http" json:"http"`
}
