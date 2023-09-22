package decode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	// Shared configs
	RouterAddress   string              `mapstructure:"routerAddress"`
	ExecutorAddress string              `mapstructure:"executorAddress"`
	ChainID         valueobject.ChainID `mapstructure:"chainId"`

	// L2 encode configs
	UseL2Optimize             bool            `mapstructure:"useL2Optimize"`
	FunctionSelectorMappingID map[string]byte `mapstructure:"functionSelectorMappingID"`
}
