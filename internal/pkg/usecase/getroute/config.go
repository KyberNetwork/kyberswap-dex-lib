package getroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID                 valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress           string              `mapstructure:"routerAddress" json:"routerAddress"`
	ExecutorAddress         string              `mapstructure:"executorAddress" json:"executorAddress"`
	GasTokenAddress         string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources        []string            `mapstructure:"availableSources" json:"availableSources"`
	UnscalableSources       []string            `mapstructure:"unscalableSources" json:"unscalableSources"`
	ExcludedSourcesByClient map[string][]string `mapstructure:"excludedSourcesByClient" json:"excludedSourcesByClient"`

	Aggregator        AggregatorConfig                        `mapstructure:"aggregator" json:"aggregator"`
	Cache             valueobject.CacheConfig                 `mapstructure:"cache" json:"cache"`
	SafetyQuoteConfig *valueobject.SafetyQuoteReductionConfig `mapstructure:"safetyQuoteConfig" json:"safetyQuoteConfig"`
	CorrelatedPairs   map[string]string                       `mapstructure:"correlatedPairs" json:"correlatedPairs"`
	DefaultPoolsIndex string                                  `mapstructure:"defaultPoolsIndex" json:"defaultPoolsIndex"`
	Salt              string                                  `mapstructure:"salt" json:"salt"`
}

type AggregatorConfig struct {
	WhitelistedTokenSet map[string]bool                 `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	FinderOptions       valueobject.FinderOptions       `mapstructure:"finderOptions" json:"finderOptions"`
	FeatureFlags        valueobject.FeatureFlags        `mapstructure:"featureFlags"`

	DexUseAEVM map[string]bool `mapstructure:"dexUseAEVM"`
}
