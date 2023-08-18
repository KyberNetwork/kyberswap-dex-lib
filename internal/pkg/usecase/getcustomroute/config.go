package getcustomroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID         valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress   string              `mapstructure:"routerAddress" json:"routerAddress"`
	GasTokenAddress string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
}
