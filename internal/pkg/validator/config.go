package validator

type Config struct {
	BuildRouteParams     BuildRouteParamsConfig     `mapstructure:"buildRouteParams"`
	GetRouteEncodeParams GetRouteEncodeParamsConfig `mapstructure:"getRouteEncodeParams"`
}

type BuildRouteParamsConfig struct {
	SlippageToleranceLTE int64 `mapstructure:"slippageToleranceLte"`
	SlippageToleranceGTE int64 `mapstructure:"slippageToleranceGte"`
}

type GetRouteEncodeParamsConfig struct {
	SlippageToleranceLTE int64 `mapstructure:"slippageToleranceLte"`
	SlippageToleranceGTE int64 `mapstructure:"slippageToleranceGte"`
}
