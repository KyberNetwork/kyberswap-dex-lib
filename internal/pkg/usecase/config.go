package usecase

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	PoolFactory factory.PoolFactoryConfig `mapstructure:"poolFactory"`
	CacheRoute  CacheRouteConfig          `mapstructure:"cacheRoute"`
	GetRoutes   GetRoutesConfig           `mapstructure:"getRoutes"`
}

type (
	GetRoutesConfig struct {
		ChainID             valueobject.ChainID `mapstructure:"chainId"`
		GasTokenAddress     string              `mapstructure:"gasTokenAddress"`
		RouterAddress       string              `mapstructure:"routerAddress"`
		Epsilon             float64             `mapstructure:"epsilon"`
		BaseGas             int64               `mapstructure:"baseGas"`
		GetBestPoolsOptions GetBestPoolsOptions `mapstructure:"getBestPoolsOptions"`
		SPFAFinderOptions   SPFAFinderOptions   `mapstructure:"spfaFinderOptions"`

		EnabledDexes      []string                       `json:"enabledDexes"`
		BlacklistedPools  []string                       `mapstructure:"blacklistedPools"`
		FeatureFlags      valueobject.FeatureFlags       `mapstructure:"featureFlags"`
		WhitelistedTokens []valueobject.WhitelistedToken `mapstructure:"whitelistTokens"`
	}

	// GetBestPoolsOptions contains options getting best pools for finding route
	GetBestPoolsOptions struct {
		// DirectPoolsCount max number of pools with tokenIn-tokenOut swap by reserve in USD
		DirectPoolsCount int64 `mapstructure:"directPoolsCount"`

		// WhitelistPoolsCount max number of pools with whitelist-whitelist swap by reserve in USD
		WhitelistPoolsCount int64 `mapstructure:"whitelistPoolsCount"`

		// TokenInPoolsCount max number of pools with whitelist-tokenIn swap by reserve in USD
		TokenInPoolsCount int64 `mapstructure:"tokenInPoolsCount"`

		// WhitelistPoolsCount max number of pools with whitelist-tokenOut swap by reserve in USD
		TokenOutPoolCount int64 `mapstructure:"tokenOutPoolCount"`

		// AmplifiedTvlDirectPoolsCount max number of pools with tokenIn-tokenOut swap by amplified TVL
		AmplifiedTvlDirectPoolsCount int64 `mapstructure:"amplifiedTvlDirectPoolsCount"`

		// AmplifiedTvlWhitelistPoolsCount max number of pools with whitelist-whitelist swap by amplified TVL
		AmplifiedTvlWhitelistPoolsCount int64 `mapstructure:"amplifiedTvlWhitelistPoolsCount"`

		// AmplifiedTvlTokenInPoolsCount max number of pools with whitelist-tokenIn swap by amplified TVL
		AmplifiedTvlTokenInPoolsCount int64 `mapstructure:"amplifiedTvlTokenInPoolsCount"`

		// AmplifiedTvlTokenOutPoolCount max number of pools with whitelist-tokenOut swap by amplified TVL
		AmplifiedTvlTokenOutPoolCount int64 `mapstructure:"amplifiedTvlTokenOutPoolCount"`
	}

	SPFAFinderOptions struct {
		MaxHops    int     `mapstructure:"maxHops"`
		MinPartUSD float64 `mapstructure:"minPartUsd"`
	}
)

type (
	BuildRouteConfig struct {
		ChainID valueobject.ChainID `mapstructure:"chainId"`
	}
)

type (
	CacheRouteConfig struct {
		CachePoints     []CachePointConfig `mapstructure:"cachePoints"`
		CacheRanges     []CacheRangeConfig `mapstructure:"cacheRanges"`
		KeyPrefix       string             `mapstructure:"keyPrefix"`
		DefaultCacheTTL time.Duration      `mapstructure:"defaultCacheTTL"`
	}

	CachePointConfig struct {
		Amount int64         `mapstructure:"amount"`
		TTL    time.Duration `mapstructure:"ttl"`
	}

	CacheRangeConfig struct {
		FromUSD int           `mapstructure:"fromUsd"`
		ToUSD   int           `mapstructure:"toUsd"`
		TTL     time.Duration `mapstructure:"ttl"`
	}
)
