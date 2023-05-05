package route

import (
	"time"
)

type Config struct {
	RedisCache RedisCacheRepositoryConfig `mapstructure:"redisCache"`
}

type RedisCacheRepositoryConfig struct {
	Prefix         string        `mapstructure:"prefix"`
	LocalCacheSize int           `mapstructure:"localCacheSize"`
	LocalCacheTTL  time.Duration `mapstructure:"localCacheTtl"`
}
