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
	Redis   RedisRepositoryConfig   `mapstructure:"redis"`
	GoCache GoCacheRepositoryConfig `mapstructure:"goCache"`
	Http    HttpConfig              `mapstructure:"http"`
}

type RedisRepositoryConfig struct {
	Prefix string `mapstructure:"prefix"`
}

type GoCacheRepositoryConfig struct {
	Expiration      time.Duration       `mapstructure:"expiration"`
	CleanupInterval time.Duration       `mapstructure:"cleanupInterval"`
	ChainID         valueobject.ChainID `mapstruct:"chainId"`
}
