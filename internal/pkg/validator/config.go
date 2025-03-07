package validator

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

type Config struct {
	SlippageValidatorConfig SlippageValidatorConfig    `mapstructure:"slippageValidatorConfig"`
	BuildRouteParams        BuildRouteParamsConfig     `mapstructure:"buildRouteParams"`
	GetRouteEncodeParams    GetRouteEncodeParamsConfig `mapstructure:"getRouteEncodeParams"`
}

type BuildRouteParamsConfig struct {
	BlacklistedRecipientSet map[string]bool          `mapstructure:"blacklistedRecipientSet"`
	FeatureFlags            valueobject.FeatureFlags `mapstructure:"featureFlags"`
}

type GetRouteEncodeParamsConfig struct {
	BlacklistedRecipientSet map[string]bool          `mapstructure:"blacklistedRecipientSet"`
	FeatureFlags            valueobject.FeatureFlags `mapstructure:"featureFlags"`
}

type SlippageValidatorConfig struct {
	SlippageToleranceGTE float64 `mapstructure:"slippageToleranceGte"`
	SlippageToleranceLTE float64 `mapstructure:"slippageToleranceLte"`
}
