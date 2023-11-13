package valueobject

import "time"

type (
	RemoteConfig struct {
		Hash                  string              `json:"hash"`
		AvailableSources      []Source            `json:"availableSources"`
		WhitelistedTokens     []WhitelistedToken  `json:"whitelistedTokens"`
		FeatureFlags          FeatureFlags        `json:"featureFlags"`
		BlacklistedPools      []string            `json:"blacklistedPools"`
		Log                   Log                 `json:"log"`
		GetBestPoolsOptions   GetBestPoolsOptions `json:"getBestPoolsOptions"`
		FinderOptions         FinderOptions       `json:"finderOptions"`
		PregenFinderOptions   FinderOptions       `json:"pregenFinderOptions"`
		CacheConfig           CacheConfig         `json:"cache"`
		BlacklistedRecipients []string            `json:"blacklistedRecipients"`
		L2EncodePartners      []string            `json:"l2EncodePartners"`
	}

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
		IsPathGeneratorEnabled bool `mapstructure:"isPathGeneratorEnabled" json:"isPathGeneratorEnabled"`
		IsGasEstimatorEnabled  bool `mapstructure:"isGasEstimatorEnabled" json:"isGasEstimatorEnabled"`
		IsBlackjackEnabled     bool `mapstructure:"isBlackjackEnabled" json:"isBlackjackEnabled"`
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

	CacheConfig struct {
		// DefaultTTL default time to live of the cache
		DefaultTTL time.Duration `mapstructure:"defaultTtl" json:"defaultTtl"`

		// TTLByAmount time to live by amount
		// key is amount without decimals
		TTLByAmount []CachePoint `mapstructure:"ttlByAmount" json:"ttlByAmount"`

		// TTLByAmountUSDRange time to live by amount usd range
		// key is lower bound of the range
		TTLByAmountUSDRange []CacheRange `mapstructure:"ttlByAmountUsdRange" json:"ttlByAmountUsdRange"`

		PriceImpactThreshold float64 `mapstructure:"priceImpactThreshold" json:"priceImpactThreshold"`

		ShrinkFuncName       string  `mapstructure:"shrinkFuncName" json:"shrinkFuncName"`
		ShrinkFuncPowExp     float64 `mapstructure:"shrinkFuncPowExp" json:"shrinkFuncPowExp"`
		ShrinkDecimalBase    float64 `mapstructure:"shrinkDecimalBase" json:"shrinkDecimalBase"`
		ShrinkFuncLogPercent float64 `mapstructure:"shrinkFuncLogPercent" json:"shrinkFuncLogPercent"`
	}

	CachePoint struct {
		Amount float64       `mapstructure:"amount" json:"amount"`
		TTL    time.Duration `mapstructure:"ttl" json:"ttl"`
	}

	CacheRange struct {
		AmountUSDLowerBound float64       `mapstructure:"amountUSDLowerBound" json:"amountUSDLowerBound"`
		TTL                 time.Duration `mapstructure:"ttl" json:"ttl"`
	}
)

func (c CacheConfig) Equals(other CacheConfig) bool {
	if c.DefaultTTL != other.DefaultTTL ||
		c.PriceImpactThreshold != other.PriceImpactThreshold ||
		c.ShrinkFuncName != other.ShrinkFuncName ||
		c.ShrinkFuncPowExp != other.ShrinkFuncPowExp ||
		c.ShrinkFuncLogPercent != other.ShrinkFuncLogPercent {
		return false
	}

	if len(c.TTLByAmount) != len(other.TTLByAmount) {
		return false
	}

	for i, point := range c.TTLByAmount {
		if point != other.TTLByAmount[i] {
			return false
		}
	}

	if len(c.TTLByAmountUSDRange) != len(other.TTLByAmountUSDRange) {
		return false
	}

	for i, rangeItem := range c.TTLByAmountUSDRange {
		if rangeItem != other.TTLByAmountUSDRange[i] {
			return false
		}
	}

	return true
}
