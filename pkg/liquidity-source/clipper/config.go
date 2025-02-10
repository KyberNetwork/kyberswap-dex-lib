package clipper

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL      string                `mapstructure:"base_url" json:"baseUrl"`
	Timeout      durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount   int                   `mapstructure:"retry_count" json:"retryCount"`
	BasicAuthKey string                `mapstructure:"basic_auth_key" json:"basicAuthKey"`
}
