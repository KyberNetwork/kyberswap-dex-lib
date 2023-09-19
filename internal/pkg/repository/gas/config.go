package gas

import "time"

type Config struct {
	Redis     RedisRepositoryConfig `mapstructure:"redis"`
	Ristretto RistrettoConfig       `mapstructure:"ristretto"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}

type RistrettoConfig struct {
	NumCounters int64 `mapstructure:"numCounters"`
	MaxCost     int64 `mapstructure:"maxCost"`
	BufferItems int64 `mapstructure:"bufferItems"`

	SuggestedGasPrice struct {
		Cost int64         `mapstructure:"cost"`
		TTL  time.Duration `mapstructure:"ttl"`
	} `mapstructure:"suggestedGasPrice"`
}
