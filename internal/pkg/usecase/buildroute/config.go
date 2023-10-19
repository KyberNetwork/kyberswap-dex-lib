package buildroute

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

type (
	Config struct {
		ChainID                valueobject.ChainID `mapstructure:"chainId"`
		RFQ                    []RFQConfig         `mapstructure:"rfq"`
		L2EncodePartners       map[string]struct{}
		UseL2OptimizeByDefault bool                     `mapstructure:"useL2OptimizeByDefault"`
		FeatureFlags           valueobject.FeatureFlags `mapstructure:"featureFlags"`
	}
	RFQConfig struct {
		Id         string                 `mapstructure:"id"`
		Handler    string                 `mapstructure:"handler"`
		Properties map[string]interface{} `mapstructure:"properties"`
	}
)
