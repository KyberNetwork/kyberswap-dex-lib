package onchainprice

import (
	"time"
)

type Config struct {
	Grpc      GrpcConfig      `mapstructure:"grpc"`
	Ristretto RistrettoConfig `mapstructure:"ristretto"`
}

type GrpcConfig struct {
	BaseURL  string        `mapstructure:"base_url"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Insecure bool          `mapstructure:"insecure"`
	ClientID string        `mapstructure:"client_id"`
}

type RistrettoConfig struct {
	NumCounters int64 `mapstructure:"numCounters"`
	MaxCost     int64 `mapstructure:"maxCost"`
	BufferItems int64 `mapstructure:"bufferItems"`

	Price struct {
		Cost int64         `mapstructure:"cost"`
		TTL  time.Duration `mapstructure:"ttl"`
	} `mapstructure:"price"`
}
