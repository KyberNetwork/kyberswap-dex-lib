package token

import (
	"time"
)

type Config struct {
	Redis   RedisRepositoryConfig   `mapstructure:"redis"`
	GoCache GoCacheRepositoryConfig `mapstructure:"goCache"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}

type GoCacheRepositoryConfig struct {
	Expiration      time.Duration `mapstructure:"expiration"`
	CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
}
