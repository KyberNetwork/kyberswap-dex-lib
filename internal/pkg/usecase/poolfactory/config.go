package poolfactory

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID        valueobject.ChainID          `mapstructure:"chainId"`
	UseAEVM        bool                         `mapstructure:"useAEVM"`
	DexUseAEVM     map[string]bool              `mapstructure:"dexUseAEVM"`
	AddressesByDex map[string]map[string]string `mapstructure:"addressesByDex"`
}
