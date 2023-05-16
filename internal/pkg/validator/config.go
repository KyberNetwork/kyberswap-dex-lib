package validator

type Config struct {
	BuildRouteParams BuildRouteParamsConfig `mapstructure:"buildRouteParams"`
}

type BuildRouteParamsConfig struct {
	SlippageToleranceLTE int64 `mapstructure:"slippageToleranceLte"`
	SlippageToleranceGTE int64 `mapstructure:"slippageToleranceGte"`
}
