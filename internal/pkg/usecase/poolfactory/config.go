package poolfactory

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID valueobject.ChainID `mapstructure:"chainId"`
}
