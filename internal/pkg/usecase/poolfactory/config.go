package poolfactory

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID    valueobject.ChainID `mapstructure:"chainId"`
	UseAEVM    bool                `mapstructure:"useAEVM"`    // use either aevm or rpc pools
	DexUseAEVM map[string]bool     `mapstructure:"dexUseAEVM"` // use either aevm or rpc pools
}
