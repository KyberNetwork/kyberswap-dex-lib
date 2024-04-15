package onchainprice

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
)

type Config struct {
	Enabled   bool                  `mapstructure:"enabled" default:"false"`
	Grpc      GrpcConfig            `mapstructure:"grpc"`
	Ristretto price.RistrettoConfig `mapstructure:"ristretto"`
}

type GrpcConfig struct {
	BaseURL  string        `mapstructure:"base_url"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Insecure bool          `mapstructure:"insecure"`
	ClientID string        `mapstructure:"client_id"`
}
