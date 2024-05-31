package pool

import "time"

type Config struct {
	Redis     RedisRepositoryConfig `mapstructure:"redis"`
	Ristretto RistrettoConfig       `mapstructure:"ristretto"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
	// using for pagination getting faulty pool list
	MaxFaultyPoolSize int64 `mapstructure:"maxFaultyPoolSize"`
}

type RistrettoConfig struct {
	NumCounters int64 `mapstructure:"numCounters"`
	MaxCost     int64 `mapstructure:"maxCost"`
	BufferItems int64 `mapstructure:"bufferItems"`

	FaultyPools struct {
		Cost int64         `mapstructure:"cost"`
		TTL  time.Duration `mapstructure:"ttl"`
	} `mapstructure:"faultyPools"`
}
