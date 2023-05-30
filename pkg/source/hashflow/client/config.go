package client

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/duration"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	HTTP       HTTPConfig       `mapstructure:"http"`
	RedisCache RedisCacheConfig `mapstructure:"redisCache"`
}

type HTTPConfig struct {
	ChainID    valueobject.ChainID `json:"chainID"`
	BaseURL    string              `mapstructure:"baseUrl"`
	APIKey     string              `mapstructure:"apiKey"`
	Source     string              `mapstructure:"source"`
	Timeout    duration.Duration   `mapstructure:"timeout"`
	RetryCount int                 `mapstructure:"retryCount"`
}

type RedisCacheConfig struct {
	Prefix    string            `mapstructure:"prefix"`
	Separator string            `mapstructure:"separator"`
	TTL       duration.Duration `mapstructure:"ttl"`
}
