package encode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	RouterAddress   string              `mapstructure:"routerAddress"`
	ExecutorAddress string              `mapstructure:"executorAddress"`
	KyberLOAddress  string              `mapstructure:"kyberLOAddress"`
	ChainID         valueobject.ChainID `mapstructure:"chainId"`
}
