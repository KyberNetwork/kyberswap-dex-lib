package buildroute

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Config struct {
		ChainID           valueobject.ChainID      `mapstructure:"chainId"`
		RFQ               []RFQConfig              `mapstructure:"rfq"`
		FeatureFlags      valueobject.FeatureFlags `mapstructure:"featureFlags"`
		FaultyPoolsConfig FaultyPoolsConfig        `mapstructure:"faultyPools"`
	}
	RFQConfig struct {
		Id         string                 `mapstructure:"id"`
		Handler    string                 `mapstructure:"handler"`
		Properties map[string]interface{} `mapstructure:"properties"`
	}

	FaultyPoolsConfig struct {
		// For the sake of simplicity, we should set window size >= 1 minute,
		// otherwise we have to modify window keys on Redis, adding seconds value to the keys.
		WindowSize        time.Duration `mapstructure:"windowSize" json:"windowSize"`
		FaultyExpiredTime time.Duration `mapstructure:"faultyExpiredTime" json:"faultyExpiredTime"`
		// Min slippage threshold configured in BPS format, ex: 0.01% -> 1, 0.5% -> 50
		MinSlippageThreshold int64           `mapstructure:"minSlippageThreshold" json:"minSlippageThreshold"`
		WhitelistedTokenSet  map[string]bool `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	}
)
