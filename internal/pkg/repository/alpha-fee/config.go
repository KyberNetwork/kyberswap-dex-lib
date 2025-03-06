package route

import "time"

type Config struct {
	Redis RedisRepositoryConfig `mapstructure:"redis"`
}

type RedisRepositoryConfig struct {
	Prefix    string        `mapstructure:"prefix"`
	Separator string        `mapstructure:"separator"`
	TTL       time.Duration `mapstructure:"ttl"`
}
