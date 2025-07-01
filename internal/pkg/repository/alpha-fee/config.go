package route

import "time"

type Config struct {
	Redis RedisRepositoryConfig `mapstructure:"redis"`
}

type RedisRepositoryConfig struct {
	Prefix    string        `mapstructure:"prefix" json:"prefix"`
	Separator string        `mapstructure:"separator" json:"separator"`
	TTL       time.Duration `mapstructure:"ttl" json:"ttl"`
}
