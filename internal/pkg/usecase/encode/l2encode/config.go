package l2encode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	RouterAddress             string              `mapstructure:"routerAddress"`
	ExecutorAddress           string              `mapstructure:"executorAddress"`
	ChainID                   valueobject.ChainID `mapstructure:"chainId"`
	IsPositiveSlippageEnabled bool                `mapstructure:"isPositiveSlippageEnabled"`
	MinimumPSThreshold        int64               `mapstructure:"minimumPSThreshold"`
	FunctionSelectorMappingID map[string]byte     `mapstructure:"functionSelectorMappingID"`
	ExecutorAddressByClientID map[string]string   `mapstructure:"executorAddressByClientID"`
}
