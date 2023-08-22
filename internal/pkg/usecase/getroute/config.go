package getroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID          valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress    string              `mapstructure:"routerAddress" json:"routerAddress"`
	GasTokenAddress  string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources []string            `mapstructure:"availableSources" json:"availableSources"`

	AmmAggregator AmmAggregatorConfig     `mapstructure:"ammAggregator" json:"ammAggregator"`
	Cache         valueobject.CacheConfig `mapstructure:"cache" json:"cache"`
}

type AmmAggregatorConfig struct {
	WhitelistedTokenSet map[string]bool                 `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	FinderOptions       valueobject.FinderOptions       `mapstructure:"finderOptions" json:"finderOptions"`
}
