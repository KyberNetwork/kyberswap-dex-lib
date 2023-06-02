package client

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/duration"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	HTTP        HTTPConfig        `mapstructure:"http"`
	RedisCache  RedisCacheConfig  `mapstructure:"redis_cache" json:"redis_cache"`
	MemoryCache MemoryCacheConfig `mapstructure:"memory_cache" json:"memory_cache"`
}

type HTTPConfig struct {
	ChainID    valueobject.ChainID `json:"chain_id"`
	BaseURL    string              `mapstructure:"base_url" json:"base_url"`
	APIKey     string              `mapstructure:"api_key" json:"api_key"`
	Source     string              `mapstructure:"source"`
	Timeout    duration.Duration   `mapstructure:"timeout"`
	RetryCount int                 `mapstructure:"retry_count" json:"retry_count"`
}

type RedisCacheConfig struct {
	Prefix    string            `mapstructure:"prefix"`
	Separator string            `mapstructure:"separator"`
	TTL       duration.Duration `mapstructure:"ttl"`
}

type MemoryCacheConfig struct {
	TTL duration.Duration `mapstructure:"ttl"`
}
