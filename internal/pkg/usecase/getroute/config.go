package getroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils/token"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID          valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress    string              `mapstructure:"routerAddress" json:"routerAddress"`
	GasTokenAddress  string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources []string            `mapstructure:"availableSources" json:"availableSources"`

	Aggregator        AggregatorConfig                        `mapstructure:"aggregator" json:"aggregator"`
	Cache             valueobject.CacheConfig                 `mapstructure:"cache" json:"cache"`
	SafetyQuoteConfig *valueobject.SafetyQuoteReductionConfig `mapstructure:"safetyQuoteConfig" json:"safetyQuoteConfig"`
	CorrelatedPairs   map[string]string                       `mapstructure:"correlatedPairs" json:"correlatedPairs"`
}

type AggregatorConfig struct {
	WhitelistedTokenSet map[string]bool                 `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	FinderOptions       valueobject.FinderOptions       `mapstructure:"finderOptions" json:"finderOptions"`
	FeatureFlags        valueobject.FeatureFlags        `mapstructure:"featureFlags"`

	TokensThresholdForOnchainPrice uint32          `mapstructure:"tokensThresholdForOnchainPrice" json:"tokensThresholdForOnchainPrice"`
	DexUseAEVM                     map[string]bool `mapstructure:"dexUseAEVM"`
}

func (cfg *AggregatorConfig) CheckTokenThreshold(address string) bool {
	return token.CheckTokenThreshold(address, cfg.TokensThresholdForOnchainPrice)
}
