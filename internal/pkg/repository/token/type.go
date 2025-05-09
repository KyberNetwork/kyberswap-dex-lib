package token

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	KeyTokens    = "tokens"
	KeyTokenInfo = "tokenInfo"
)

type Config struct {
	Redis   RedisRepositoryConfig `mapstructure:"redis"`
	GoCache *RistrettoConfig      `mapstructure:"ristretto"`
	Http    HttpConfig            `mapstructure:"http"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}

type RistrettoConfig struct {
	ChainID             valueobject.ChainID `mapstruct:"chainId"`
	WhitelistedTokenSet map[string]bool     `mapstructure:"whitelistedTokenSet"`

	Token struct {
		Cost        int64         `mapstructure:"cost"`
		NumCounters int64         `mapstructure:"numCounters"`
		MaxCost     int64         `mapstructure:"maxCost"`
		BufferItems int64         `mapstructure:"bufferItems"`
		TTL         time.Duration `mapstructure:"ttl"`
	} `mapstructure:"token"`

	TokenInfo struct {
		Cost        int64 `mapstructure:"cost"`
		NumCounters int64 `mapstructure:"numCounters"`
		MaxCost     int64 `mapstructure:"maxCost"`
		BufferItems int64 `mapstructure:"bufferItems"`
	} `mapstructure:"tokenInfo"`
}
