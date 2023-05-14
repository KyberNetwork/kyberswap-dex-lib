package poolfactory

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

type Config struct {
	ChainID valueobject.ChainID `mapstructure:"chainId"`
}
