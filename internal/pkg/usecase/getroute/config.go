package getroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID                 valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress           string              `mapstructure:"routerAddress" json:"routerAddress"`
	KyberExecutorAddress    string
	GasTokenAddress         string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources        []string            `mapstructure:"availableSources" json:"availableSources"`
	UnscalableSources       []string            `mapstructure:"unscalableSources" json:"unscalableSources"`
	ScaleHelperClients      []string            `mapstructure:"scaleHelperClients" json:"scaleHelperClients"`
	ExcludedSourcesByClient map[string][]string `mapstructure:"excludedSourcesByClient" json:"excludedSourcesByClient"`

	Aggregator        AggregatorConfig                        `mapstructure:"aggregator" json:"aggregator"`
	Cache             valueobject.CacheConfig                 `mapstructure:"cache" json:"cache"`
	SafetyQuoteConfig *valueobject.SafetyQuoteReductionConfig `mapstructure:"safetyQuoteConfig" json:"safetyQuoteConfig"`
	AlphaFeeConfig    valueobject.AlphaFeeConfig              `mapstructure:"alphaFeeConfig" json:"alphaFeeConfig"`
	CorrelatedPairs   map[string]string                       `mapstructure:"correlatedPairs" json:"correlatedPairs"`
	DefaultPoolsIndex string                                  `mapstructure:"defaultPoolsIndex" json:"defaultPoolsIndex"`
	Salt              string                                  `mapstructure:"salt" json:"salt"`

	FeatureFlags valueobject.FeatureFlags `mapstructure:"featureFlags" json:"featureFlags"`
}

type AggregatorConfig struct {
	WhitelistedTokenSet map[string]bool                 `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	FinderOptions       valueobject.FinderOptions       `mapstructure:"finderOptions" json:"finderOptions"`
	FeatureFlags        valueobject.FeatureFlags        `mapstructure:"featureFlags"`

	DexUseAEVM map[string]bool `mapstructure:"dexUseAEVM"`
}
