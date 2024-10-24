package clipper

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type HTTPClientConfig struct {
	BaseURL      string                `mapstructure:"base_url" json:"base_url"`
	Timeout      durationjson.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount   int                   `mapstructure:"retry_count" json:"retry_count"`
	BasicAuthKey string                `mapstructure:"basic_auth_key" json:"basic_auth_key"`
}
