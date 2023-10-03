package pathgenerator

import "github.com/KyberNetwork/router-service/internal/pkg/usecase/generatepath"

type Config struct {
	Redis RedisRepositoryConfig `mapstructure:"redis"`
}

type RedisRepositoryConfig struct {
	Prefix string
}

type PregenTokenAmounts struct {
	TokenAmounts []generatepath.TokenAndAmounts `json:"tokenAmounts"`
	Timestamp    int64                          `json:"timestamp"`
}

const PregenTokenAmountsKey = "pregen_amounts"
