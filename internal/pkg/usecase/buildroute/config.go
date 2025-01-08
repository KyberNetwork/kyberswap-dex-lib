package buildroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Config struct {
		ChainID                       valueobject.ChainID      `mapstructure:"chainId"`
		RFQ                           map[string]RFQConfig     `mapstructure:"rfq"`
		FeatureFlags                  valueobject.FeatureFlags `mapstructure:"featureFlags"`
		FaultyPoolsConfig             FaultyPoolsConfig        `mapstructure:"faultyPools"`
		RFQAcceptableSlippageFraction int64                    `mapstructure:"rfqAcceptableSlippageFraction"` // Config in BPS
		FaultyPoolDetectorDisabled    bool                     `mapstructure:"faultyPoolDetectorDisabled"`
		Salt                          string                   `mapstructure:"salt"`
	}
	RFQConfig struct {
		Handler    string                 `mapstructure:"handler"`
		Properties map[string]interface{} `mapstructure:"properties"`
	}

	FaultyPoolsConfig struct {
		// Min slippage threshold configured in BPS format, ex: 0.01% -> 1, 0.5% -> 50
		MinSlippageThreshold int64           `mapstructure:"minSlippageThreshold" json:"minSlippageThreshold"`
		WhitelistedTokenSet  map[string]bool `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	}
)
