package valueobject

import (
	"math/big"
	"time"
)

type (
	RemoteConfig struct {
		Hash                    string              `json:"hash"`
		AvailableSources        []Source            `json:"availableSources"`
		UnscalableSources       []Source            `json:"unscalableSources"`
		ExcludedSourcesByClient map[string][]Source `json:"excludedSourcesByClient"`
		DexUseAEVM              map[string]bool     `json:"dexUseAEVM"`
		WhitelistedTokens       []WhitelistedToken  `json:"whitelistedTokens"`
		FeatureFlags            FeatureFlags        `json:"featureFlags"`
		BlacklistedPools        []string            `json:"blacklistedPools"`
		Log                     Log                 `json:"log"`
		GetBestPoolsOptions     GetBestPoolsOptions `json:"getBestPoolsOptions"`
		FinderOptions           FinderOptions       `json:"finderOptions"`
		PregenFinderOptions     FinderOptions       `json:"pregenFinderOptions"`
		CacheConfig             CacheConfig         `json:"cache"`
		BlacklistedRecipients   []string            `json:"blacklistedRecipients"`

		RFQAcceptableSlippageFraction  int64                      `json:"rfqAcceptableSlippageFraction"`
		TokensThresholdForOnchainPrice uint32                     `json:"tokensThresholdForOnchainPrice"`
		FaultyPoolsConfig              FaultyPoolsConfig          `json:"faultyPoolsConfig"`
		SafetyQuoteReduction           SafetyQuoteReductionConfig `json:"safetyQuoteReduction"`
		DexalotUpscalePercent          int                        `json:"dexalotUpscalePercent"`
		AlphaFeeConfig                 AlphaFeeConfig             `json:"alphaFeeConfig"`
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
		IsHillClimbEnabled                bool `mapstructure:"isHillClimbEnabled" json:"isHillClimbEnabled"`
		IsDerivativeHillClimbEnabled      bool `mapstructure:"isDerivativeHillClimbEnabled" json:"isDerivativeHillClimbEnabled"`
		IsGasEstimatorEnabled             bool `mapstructure:"isGasEstimatorEnabled" json:"isGasEstimatorEnabled"`
		IsBlackjackEnabled                bool `mapstructure:"isBlackjackEnabled" json:"isBlackjackEnabled"`
		IsOptimizeExecutorFlagsEnabled    bool `mapstructure:"isOptimizeExecutorFlagsEnabled" json:"isOptimizeExecutorFlagsEnabled"`
		ShouldValidateSender              bool `mapstructure:"shouldValidateSender" json:"shouldValidateSender"`
		IsAEVMEnabled                     bool `mapstructure:"isAEVMEnabled" json:"isAEVMEnabled"`
		IsFaultyPoolDetectorEnable        bool `mapstructure:"isFaultyPoolDetectorEnable" json:"isFaultyPoolDetectorEnable"`
		IsLiquidityScoreIndexEnable       bool `mapstructure:"isLiquidityScoreEnable" json:"isLiquidityScoreEnable"`
		IsRouteCachedEnable               bool `mapstructure:"isRouteCachedEnable" json:"isRouteCachedEnable"`
		IsMergeDuplicateSwapEnabled       bool `mapstructure:"isMergeDuplicateSwapEnabled" json:"isMergeDuplicateSwapEnabled"`
		IsKyberPrivateLimitOrdersEnabled  bool `mapstructure:"isKyberPrivateLimitOrdersEnabled" json:"isKyberPrivateLimitOrdersEnabled"`
		IsAlphaFeeReductionEnable         bool `mapstructure:"isAlphaFeeReductionEnable" json:"isAlphaFeeReductionEnable"`
		IsHillClimbEnabledForAMMBestRoute bool `mapstructure:"isHillClimbEnabledForAMMBestRoute" json:"isHillClimbEnabledForAMMBestRoute"`
		ShouldReturnAlphaFee              bool `mapstructure:"shouldReturnAlphaFee" json:"shouldReturnAlphaFee"`
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
		// min threshold for amount in using in liquidity score index
		AmountInThreshold float64 `mapstructure:"amountInThreshold" json:"amountInThreshold"`
	}

	FinderOptions struct {
		Type                         string          `mapstructure:"type" json:"type"`
		MaxHops                      uint            `mapstructure:"maxHops" json:"maxHops"`
		DistributionPercent          uint            `mapstructure:"distributionPercent" json:"distributionPercent"`
		MaxPathsInRoute              uint            `mapstructure:"maxPathsInRoute" json:"maxPathsInRoute"`
		MaxPathsInFallbackRoute      uint            `mapstructure:"maxPathsInFallbackRoute" json:"maxPathsInFallbackRoute"`
		MaxPathsToGenerate           uint            `mapstructure:"maxPathsToGenerate" json:"maxPathsToGenerate"`
		MaxPathsToReturn             uint            `mapstructure:"maxPathsToReturn" json:"maxPathsToReturn"`
		MinPartUSD                   float64         `mapstructure:"minPartUSD" json:"minPartUSD"`
		MinThresholdAmountInUSD      float64         `mapstructure:"minThresholdAmountInUSD" json:"minThresholdAmountInUSD"`
		MaxThresholdAmountInUSD      float64         `mapstructure:"maxThresholdAmountInUSD" json:"maxThresholdAmountInUSD"`
		ExtraPathsPerNodeByTokens    map[string]uint `mapstructure:"extraPathsPerNodeByTokens" json:"extraPathsPerNodeByTokens"`
		FullAmountGeneratePathsPrice float64         `mapstructure:"fullAmountGeneratePathsPrice" json:"fullAmountGeneratePathsPrice"`

		HillClimbDistributionPercent uint32  `mapstructure:"hillClimbDistributionPercent" json:"hillClimbDistributionPercent"`
		HillClimbIteration           uint32  `mapstructure:"hillClimbIteration" json:"hillClimbIteration"`
		HillClimbMinPartUSD          float64 `mapstructure:"hillClimbMinPartUSD" json:"hillClimbMinPartUSD"`

		DerivativeHillClimbIteration        int     `mapstructure:"derivativeHillClimbIteration" json:"derivativeHillClimbIteration"`
		DerivativeHillClimbImproveThreshold float64 `mapstructure:"derivativeHillClimbImproveThreshold" json:"derivativeHillClimbImproveThreshold"`

		ScaleHelperClients []string `mapstructure:"scaleHelperClients" json:"scaleHelperClients"`

		// If true then route finding is performed remotely in AEVM server
		UseAEVMRemoteFinder bool `mapstructure:"useAEVMRemoteFinder" json:"useAEVMRemoteFinder"`
		// In AEVM server, if true then CalcAmountOut calls use AEVM pool
		RemoteUseAEVMPool bool `mapstructure:"remoteUseAEVMPool" json:"remoteUseAEVMPool"`
		// Locally, if true then CalcAmountOut calls use AEVM pool
		LocalUseAEVMPool bool `mapstructure:"localUseAEVMPool" json:"localUseAEVMPool"`
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

		TTLByAmountRange []AmountInCacheRange `mapstructure:"ttlByAmountRange" json:"ttlByAmountRange"`

		PriceImpactThreshold float64 `mapstructure:"priceImpactThreshold" json:"priceImpactThreshold"`

		// cache config for amount in usd
		ShrinkFuncName       string  `mapstructure:"shrinkFuncName" json:"shrinkFuncName"`
		ShrinkFuncPowExp     float64 `mapstructure:"shrinkFuncPowExp" json:"shrinkFuncPowExp"`
		ShrinkDecimalBase    float64 `mapstructure:"shrinkDecimalBase" json:"shrinkDecimalBase"`
		ShrinkFuncLogPercent float64 `mapstructure:"shrinkFuncLogPercent" json:"shrinkFuncLogPercent"`
		// Min amount in USD to cache, fix bug panic due to can not format number like 5e-324 to float64
		MinAmountInUSD float64 `mapstructure:"minAmountInUSD" json:"minAmountInUSD"`

		// cache config for amount in
		ShrinkAmountInConfigs   []ShrinkFunctionConfig `mapstructure:"shrinkAmountInConfigs" json:"shrinkAmountInConfigs"`
		ShrinkAmountInThreshold float64                `mapstructure:"shrinkAmountInThreshold" json:"shrinkAmountInThreshold"`

		EnableNewCacheKeyGenerator bool `mapstructure:"enableNewCacheKeyGenerator" json:"enableNewCacheKeyGenerator"`
	}

	SafetyQuoteReductionConfig struct {
		ExcludeOneSwapEnable bool               `mapstructure:"excludeOneSwapEnable" json:"excludeOneSwapEnable"`
		Factor               map[string]float64 `mapstructure:"factor" json:"factor"`
		WhitelistedClient    []string           `mapstructure:"whitelistedClient" json:"whitelistedClient"`
		// tokenGroup config doesn't need to update from remote config
		TokenGroupConfig *TokenGroupConfig `mapstructure:"tokenGroupConfig"`
	}

	TokenGroupConfig struct {
		StableGroup      map[string]bool `mapstructure:"stable"`
		CorrelatedGroup1 map[string]bool `mapstructure:"correlated-1"`
		CorrelatedGroup2 map[string]bool `mapstructure:"correlated-2"`
		CorrelatedGroup3 map[string]bool `mapstructure:"correlated-3"`
	}

	AlphaFeeConfig struct {
		ReductionConfig AlphaFeeReductionConfig `mapstructure:"reductionConfig" json:"reductionConfig"`
		TTL             time.Duration           `mapstructure:"ttl" json:"ttl"`
	}

	AlphaFeeReductionConfig struct {
		ReductionFactorInBps map[string]float64 `mapstructure:"reductionFactorInBps" json:"reductionFactorInBps"`
		// To avoid amm best path returns weird route due to lack of swap source, we must check differency between
		// amm best path and multi best path do not exeed AlphaFeeSlippageTolerance config
		MaxThresholdPercentageInBps  int64   `mapstructure:"maxThresholdPercentageInBps" json:"maxThresholdPercentageInBps"`
		MinDifferentThresholdUSD     float64 `mapstructure:"minDifferentThresholdUSD" json:"minDifferentThresholdUSD"`
		MinDifferentThresholdBps     int64   `mapstructure:"minDifferentThresholdBps" json:"minDifferentThresholdBps"`
		DefaultAlphaFeePercentageBps float64 `mapstructure:"defaultAlphaFeePercentageBps" json:"defaultAlphaFeePercentageBps"`
	}

	CachePoint struct {
		Amount float64       `mapstructure:"amount" json:"amount"`
		TTL    time.Duration `mapstructure:"ttl" json:"ttl"`
	}

	CacheRange struct {
		AmountUSDLowerBound float64       `mapstructure:"amountUSDLowerBound" json:"amountUSDLowerBound"`
		TTL                 time.Duration `mapstructure:"ttl" json:"ttl"`
	}

	AmountInCacheRange struct {
		AmountLowerBound *big.Int      `mapstructure:"amountLowerBound" json:"amountLowerBound"`
		TTL              time.Duration `mapstructure:"ttl" json:"ttl"`
	}

	ShrinkFunctionConfig struct {
		ShrinkFuncName string `mapstructure:"shrinkFuncName" json:"shrinkFuncName"`
		/** If use decimal rounding, shrink func constant will be shrinkDecimalBase
		 ** If use logarithm rounding, shrink func constant will be shrinkFuncLogPercent
		 */
		ShrinkFuncConstant float64 `mapstructure:"shrinkFuncConstant" json:"ShrinkFuncConstant"`
	}

	FaultyPoolsConfig struct {
		// Min slippage threshold configured in BPS format, ex: 0.01% -> 1, 0.5% -> 50
		MinSlippageThreshold float64 `mapstructure:"minSlippageThreshold" json:"minSlippageThreshold"`
	}
)

func (c ShrinkFunctionConfig) Equals(other ShrinkFunctionConfig) bool {
	if c.ShrinkFuncName != other.ShrinkFuncName ||
		c.ShrinkFuncConstant != other.ShrinkFuncConstant {
		return false
	}

	return true
}

func (c CacheConfig) Equals(other CacheConfig) bool {
	if c.DefaultTTL != other.DefaultTTL ||
		c.PriceImpactThreshold != other.PriceImpactThreshold ||
		c.ShrinkFuncName != other.ShrinkFuncName ||
		c.ShrinkFuncPowExp != other.ShrinkFuncPowExp ||
		c.ShrinkFuncLogPercent != other.ShrinkFuncLogPercent ||
		c.EnableNewCacheKeyGenerator != other.EnableNewCacheKeyGenerator ||
		c.ShrinkAmountInThreshold != other.ShrinkAmountInThreshold {
		return false
	}

	for i, v := range c.ShrinkAmountInConfigs {
		if v != other.ShrinkAmountInConfigs[i] {
			return false
		}
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

type finderTypes struct {
	SPFAv2            string
	RetryDynamicPools string
}

var (
	FinderTypes = finderTypes{
		SPFAv2:            "spfaV2",
		RetryDynamicPools: "retry",
	}
)
