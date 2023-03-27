package api

import (
	"time"
)

type Config struct {
	DefaultTTL time.Duration `mapstructure:"defaultTTL"`

	GetRoutes    ItemConfig `mapstructure:"getRoutes"`
	BuildRoute   ItemConfig `mapstructure:"buildRoute"`
	GetPools     ItemConfig `mapstructure:"getPools"`
	GetTokens    ItemConfig `mapstructure:"getTokens"`
	GetPublicKey ItemConfig `mapstructure:"getPublicKey"`

	GetRouteEncode ItemConfig `mapstructure:"getRouteEncode"`
}

type ItemConfig struct {
	IsCacheEnabled bool          `mapstructure:"isCacheEnabled"`
	TTL            time.Duration `mapstructure:"ttl"`

	IsTimeoutEnabled bool          `mapstructure:"isTimeoutEnabled"`
	Timeout          time.Duration `mapstructure:"timeout"`
}
