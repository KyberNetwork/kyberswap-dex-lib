package valueobject

type (
	Dex string

	WhitelistedToken struct {
		Address  string `mapstructure:"address" json:"address"`
		Name     string `mapstructure:"name" json:"name"`
		Symbol   string `mapstructure:"symbol" json:"symbol"`
		Decimals uint8  `mapstructure:"decimals" json:"decimals"`
		CgkId    string `mapstructure:"cgkId" json:"cgkId"`
	}

	// should include variable which need not to restart pods.
	FeatureFlags struct {
		UseOptimizedSPFA bool `mapstructure:"useOptimizedSPFA"`
	}

	Log struct {
		ConsoleLevel string `json:"consoleLevel"`
	}
)

type RemoteConfig struct {
	Hash              string             `json:"hash"`
	EnabledDexes      []Dex              `json:"enabledDexes"`
	WhitelistedTokens []WhitelistedToken `json:"whitelistedTokens"`
	FeatureFlags      FeatureFlags       `json:"featureFlags"`
	BlacklistedPools  []string           `json:"blacklistedPools"`
	Log               Log                `json:"log"`
}
