package encode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	// Shared configs
	ChainID                   valueobject.ChainID `mapstructure:"chainId"`
	IsPositiveSlippageEnabled bool                `mapstructure:"isPositiveSlippageEnabled"`
	MinimumPSThreshold        int64               `mapstructure:"minimumPSThreshold"`
	ExecutorAddressByClientID map[string]string   `mapstructure:"executorAddressByClientID"`

	// L1 encode configs
	RouterAddress   string `mapstructure:"routerAddress"`
	ExecutorAddress string `mapstructure:"executorAddress"`

	// L2 encode configs
	// We use L1 router for L2 encode, so no separate L2RouterAddress config
	L2ExecutorAddress         string          `mapstructure:"l2ExecutorAddress"`
	FunctionSelectorMappingID map[string]byte `mapstructure:"functionSelectorMappingID"` // Map between selector to the ID registered in the SC
}
