package buildroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Config struct {
		ChainID                       valueobject.ChainID        `mapstructure:"chainId"`
		ForceSourceByIp               map[string]string          `mapstructure:"forceSourceByIp"`
		ValidateChecksumBySource      map[string]bool            `mapstructure:"validateChecksumBySource"`
		RFQ                           map[string]RFQConfig       `mapstructure:"rfq"`
		FeatureFlags                  valueobject.FeatureFlags   `mapstructure:"featureFlags"`
		FaultyPoolsConfig             FaultyPoolsConfig          `mapstructure:"faultyPools"`
		PublisherConfig               PublisherConfig            `mapstructure:"publisher"`
		RFQAcceptableSlippageFraction int64                      `mapstructure:"rfqAcceptableSlippageFraction"` // bps
		FaultyPoolDetectorDisabled    bool                       `mapstructure:"faultyPoolDetectorDisabled"`
		AlphaFeeConfig                valueobject.AlphaFeeConfig `mapstructure:"alphaFeeConfig"`
		Salt                          string                     `mapstructure:"salt"`
		ClientRefCode                 map[string]string          `mapstructure:"clientRefCode"`
	}

	AlphaFeeConfig struct {
		DefaultAlphaFeePercentageBps float64 `mapstructure:"defaultAlphaFeePercentageBps"`
	}

	PublisherConfig struct {
		AggregatorTransactionTopic string `mapstructure:"aggregatorTransactionTopic"`
	}

	RFQConfig struct {
		Handler    string                 `mapstructure:"handler"`
		Properties map[string]interface{} `mapstructure:"properties"`
	}

	FaultyPoolsConfig struct {
		// Min slippage threshold configured in BPS format, ex: 0.01% -> 1, 0.5% -> 50
		MinSlippageThreshold float64         `mapstructure:"minSlippageThreshold" json:"minSlippageThreshold"`
		WhitelistedTokenSet  map[string]bool `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	}
)
