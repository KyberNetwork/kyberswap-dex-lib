package valueobject

type (
	Source string

	WhitelistedToken struct {
		Address  string `mapstructure:"address" json:"address"`
		Name     string `mapstructure:"name" json:"name"`
		Symbol   string `mapstructure:"symbol" json:"symbol"`
		Decimals uint8  `mapstructure:"decimals" json:"decimals"`
		CgkId    string `mapstructure:"cgkId" json:"cgkId"`
	}

	// should include variable which need not to restart pods.
	FeatureFlags struct {
	}

	Log struct {
		ConsoleLevel string `json:"consoleLevel"`
	}

	GetBestPoolsOptions struct {
		DirectPoolsCount                int64 `mapstructure:"directPoolsCount" json:"directPoolsCount"`
		WhitelistPoolsCount             int64 `mapstructure:"whitelistPoolsCount" json:"whitelistPoolsCount"`
		TokenInPoolsCount               int64 `mapstructure:"tokenInPoolsCount" json:"tokenInPoolsCount"`
		TokenOutPoolCount               int64 `mapstructure:"tokenOutPoolCount" json:"tokenOutPoolCount"`
		AmplifiedTvlDirectPoolsCount    int64 `mapstructure:"amplifiedTvlDirectPoolsCount" json:"amplifiedTvlDirectPoolsCount"`
		AmplifiedTvlWhitelistPoolsCount int64 `mapstructure:"amplifiedTvlWhitelistPoolsCount" json:"amplifiedTvlWhitelistPoolsCount"`
		AmplifiedTvlTokenInPoolsCount   int64 `mapstructure:"amplifiedTvlTokenInPoolsCount" json:"amplifiedTvlTokenInPoolsCount"`
		AmplifiedTvlTokenOutPoolCount   int64 `mapstructure:"amplifiedTvlTokenOutPoolCount" json:"amplifiedTvlTokenOutPoolCount"`
	}

	FinderOptions struct {
		MaxHops                 uint32  `mapstructure:"maxHops" json:"maxHops"`
		DistributionPercent     uint32  `mapstructure:"distributionPercent" json:"distributionPercent"`
		MaxPathsInRoute         uint32  `mapstructure:"maxPathsInRoute" json:"maxPathsInRoute"`
		MaxPathsToGenerate      uint32  `mapstructure:"maxPathsToGenerate" json:"maxPathsToGenerate"`
		MaxPathsToReturn        uint32  `mapstructure:"maxPathsToReturn" json:"maxPathsToReturn"`
		MinPartUSD              float64 `mapstructure:"minPartUSD" json:"minPartUSD"`
		MinThresholdAmountInUSD float64 `mapstructure:"minThresholdAmountInUSD" json:"minThresholdAmountInUSD"`
		MaxThresholdAmountInUSD float64 `mapstructure:"maxThresholdAmountInUSD" json:"maxThresholdAmountInUSD"`
	}
)

type RemoteConfig struct {
	Hash                string              `json:"hash"`
	AvailableSources    []Source            `json:"availableSources"`
	WhitelistedTokens   []WhitelistedToken  `json:"whitelistedTokens"`
	FeatureFlags        FeatureFlags        `json:"featureFlags"`
	BlacklistedPools    []string            `json:"blacklistedPools"`
	Log                 Log                 `json:"log"`
	GetBestPoolsOptions GetBestPoolsOptions `json:"getBestPoolsOptions"`
	FinderOptions       FinderOptions       `json:"finderOptions"`
}
