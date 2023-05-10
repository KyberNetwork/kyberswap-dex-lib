package types

// BestPools contains best scoring pool ids
type BestPools struct {
	// PoolIds pools with tokenIn-tokenOut swap
	PoolIds []string

	// WhitelistPoolIds pools with whitelist-whitelist swap
	WhitelistPoolIds []string

	// TokenInPoolIds pools with whitelist-tokenIn swap
	TokenInPoolIds []string

	// TokenOutPoolIds pools with whitelist-tokenOut swap
	TokenOutPoolIds []string
}

type GetBestPoolsOptions struct {
	// DirectPoolsCount max number of pools with tokenIn-tokenOut swap by reserve in USD
	DirectPoolsCount int64 `mapstructure:"directPoolsCount"`

	// WhitelistPoolsCount max number of pools with whitelist-whitelist swap by reserve in USD
	WhitelistPoolsCount int64 `mapstructure:"whitelistPoolsCount"`

	// TokenInPoolsCount max number of pools with whitelist-tokenIn swap by reserve in USD
	TokenInPoolsCount int64 `mapstructure:"tokenInPoolsCount"`

	// WhitelistPoolsCount max number of pools with whitelist-tokenOut swap by reserve in USD
	TokenOutPoolCount int64 `mapstructure:"tokenOutPoolCount"`

	// AmplifiedTvlDirectPoolsCount max number of pools with tokenIn-tokenOut swap by amplified TVL
	AmplifiedTvlDirectPoolsCount int64 `mapstructure:"amplifiedTvlDirectPoolsCount"`

	// AmplifiedTvlWhitelistPoolsCount max number of pools with whitelist-whitelist swap by amplified TVL
	AmplifiedTvlWhitelistPoolsCount int64 `mapstructure:"amplifiedTvlWhitelistPoolsCount"`

	// AmplifiedTvlTokenInPoolsCount max number of pools with whitelist-tokenIn swap by amplified TVL
	AmplifiedTvlTokenInPoolsCount int64 `mapstructure:"amplifiedTvlTokenInPoolsCount"`

	// AmplifiedTvlTokenOutPoolCount max number of pools with whitelist-tokenOut swap by amplified TVL
	AmplifiedTvlTokenOutPoolCount int64 `mapstructure:"amplifiedTvlTokenOutPoolCount"`
}
