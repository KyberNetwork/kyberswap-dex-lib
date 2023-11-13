package validator

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

type Config struct {
	BuildRouteParams     BuildRouteParamsConfig     `mapstructure:"buildRouteParams"`
	GetRouteEncodeParams GetRouteEncodeParamsConfig `mapstructure:"getRouteEncodeParams"`
}

type BuildRouteParamsConfig struct {
	SlippageToleranceLTE    int64                    `mapstructure:"slippageToleranceLte"`
	SlippageToleranceGTE    int64                    `mapstructure:"slippageToleranceGte"`
	BlacklistedRecipientSet map[string]bool          `mapstructure:"blacklistedRecipientSet"`
	FeatureFlags            valueobject.FeatureFlags `mapstructure:"featureFlags"`
}

type GetRouteEncodeParamsConfig struct {
	SlippageToleranceLTE    int64           `mapstructure:"slippageToleranceLte"`
	SlippageToleranceGTE    int64           `mapstructure:"slippageToleranceGte"`
	BlacklistedRecipientSet map[string]bool `mapstructure:"blacklistedRecipientSet"`
}
