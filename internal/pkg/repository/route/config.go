package route

import (
	"time"
)

type Config struct {
	Redis           RedisRepositoryConfig `mapstructure:"redis"`
	RistrettoConfig RistrettoConfig       `mapstructure:"ristretto"`
}

type RedisRepositoryConfig struct {
	Prefix string
}

type RistrettoConfig struct {
	NumCounters int64  `mapstructure:"numCounters"`
	MaxCost     int64  `mapstructure:"maxCost"`
	BufferItems int64  `mapstructure:"bufferItems"`
	Prefix      string `mapstructure:"prefix"`

	Route struct {
		Cost int64         `mapstructure:"cost"`
		TTL  time.Duration `mapstructure:"ttl"`
	} `mapstructure:"route"`
}
