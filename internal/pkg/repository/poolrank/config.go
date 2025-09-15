package poolrank

import (
	"time"
)

type Config struct {
	Redis                 RedisRepositoryConfig `mapstructure:"redis"`
	SetsNeededTobeIndexed map[string]bool       `mapstructure:"setsNeededTobeIndexed"`
	Ristretto             RistrettoConfig       `mapstructure:"ristretto"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}

type RistrettoConfig struct {
	NumCounters int64  `mapstructure:"numCounters"`
	MaxCost     int64  `mapstructure:"maxCost"`
	BufferItems int64  `mapstructure:"bufferItems"`
	Prefix      string `mapstructure:"prefix"`

	IndexCardinality struct {
		Cost int64         `mapstructure:"cost"`
		TTL  time.Duration `mapstructure:"ttl"`
	} `mapstructure:"indexCardinality"`
}
