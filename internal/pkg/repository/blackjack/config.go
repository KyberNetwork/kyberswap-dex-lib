package blackjack

import "time"

type Config struct {
	GRPCClient GRPCClientConfig `json:"grpcClient" mapstructure:"grpcClient"`
}

type GRPCClientConfig struct {
	BaseURL  string        `json:"baseUrl" mapstructure:"baseUrl"`
	Timeout  time.Duration `json:"timeout" mapstructure:"timeout"`
	Insecure bool          `json:"insecure" mapstructure:"insecure"`
	ClientID string        `json:"clientId" mapstructure:"clientId"`
}
