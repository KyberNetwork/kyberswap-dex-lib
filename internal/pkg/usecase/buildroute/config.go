package buildroute

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Config struct {
		ChainID                       valueobject.ChainID                `mapstructure:"chainId"`
		ForceSourceByIp               map[string]string                  `mapstructure:"forceSourceByIp"`
		ValidateChecksumBySource      map[string]bool                    `mapstructure:"validateChecksumBySource"`
		RFQ                           map[valueobject.Exchange]RFQConfig `mapstructure:"rfq"`
		FeatureFlags                  valueobject.FeatureFlags           `mapstructure:"featureFlags"`
		FaultyPoolsConfig             FaultyPoolsConfig                  `mapstructure:"faultyPools"`
		PublisherConfig               PublisherConfig                    `mapstructure:"publisher"`
		RFQAcceptableSlippageFraction int64                              `mapstructure:"rfqAcceptableSlippageFraction"` // bps
		FaultyPoolDetectorDisabled    bool                               `mapstructure:"faultyPoolDetectorDisabled"`
		AlphaFeeConfig                valueobject.AlphaFeeConfig         `mapstructure:"alphaFeeConfig"`
		Salt                          string                             `mapstructure:"salt"`
		ClientRefCode                 map[string]string                  `mapstructure:"clientRefCode"`
		TokenGroups                   *valueobject.TokenGroupConfig      `mapstructure:"tokenGroups"`
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
		WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
		ExpireTime          time.Duration   `mapstructure:"expireTime" json:"expireTime"`
		// SlippageConfig defines slippage settings for different token groups (stable, correlated, default)
		SlippageConfigByGroup map[string]SlippageGroupConfig `mapstructure:"slippageConfigByGroup" json:"slippageConfigByGroup"`
	}

	// SlippageGroupConfig defines slippage settings for a specific token pair type
	SlippageGroupConfig struct {
		// Buffer is added to actual slippage to protect users from price fluctuations
		// Example: if actual slippage is 1% (100) and buffer is 0.5% (50), suggested slippage will be 1.5% (150)
		Buffer float64 `mapstructure:"buffer" json:"buffer"`
		// MinThreshold is used to identify potential FOT tokens
		// Example: if threshold is 200 (2%) and actual slippage is 300 (3%), consider it might be a FOT token
		MinThreshold float64 `mapstructure:"minThreshold" json:"minThreshold"`
	}
)
