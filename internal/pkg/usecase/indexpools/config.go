package indexpools

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

type IndexPoolsConfig struct {
	WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet"`
	ChunkSize           int             `mapstructure:"chunkSize"`
	MaxGoroutines       int             `mapstructure:"maxGoroutines"`

	// If the pool has 0 TVL, and the direct index length is less than this value,
	// we will still add the pool to the indexes.
	MaxDirectIndexLenForZeroTvl int `mapstructure:"maxDirectIndexLenForZeroTvl"`
}

type TradeDataGeneratorConfig struct {
	WhitelistedTokenSet map[string]bool `mapstructure:"whitelistedTokenSet"`
	BlacklistedPoolSet  map[string]bool `mapstructure:"blacklistedPoolSet"`
	ChunkSize           int             `mapstructure:"chunkSize"`
	UseAEVM             bool            `mapstructure:"useAEVM" json:"useAEVM"`
	DexUseAEVM          map[string]bool `mapstructure:"dexUseAEVM"`
	MinDataPointNumber  int             `mapstructure:"minDataPointNumber"`
	MaxDataPointNumber  int             `mapstructure:"maxDataPointNumber"`
	AvailableSources    []string        `mapstructure:"availableSources" json:"availableSources"`
	LogError            bool            `mapstructure:"logError"`
	MaxGoroutines       int             `mapstructure:"maxGoroutines"`
	ChainName           string          `mapstructure:"chainName"`
	ExportFailedTrade   bool            `mapstructure:"exportFailedTrade"`
	FailedFileName      string          `mapstructure:"failedFileName"`
	// format will be whitelist-whitelist, token-whitelist, whitelist-token, direct
	SetsNeededTobeIndexed         map[string]bool `mapstructure:"setsNeededTobeIndexed"`
	MaxTokensLen                  int             `mapstructure:"maxTokensLen"`
	InvalidPriceImpactThreshold   float64         `mapstructure:"invalidPriceImpactThreshold"`
	PoolHasManyTokensDefaultScore float64         `mapstructure:"poolHasManyTokensDefaultScore"`
	FilePath                      string          `mapstructure:"filePath"`
	ExportZeroScores              bool            `mapstructure:"exportZeroScores"`
	minThresholdTvl               float64         `mapstructure:"minThresholdTvl"`
}

type UpdateLiquidityScoreConfig struct {
	MeanType             string                          `mapstructure:"meanType" json:"meanType"`
	GetBestPoolsOptions  valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	ChunkSize            int                             `mapstructure:"chunkSize" json:"chunkSize"`
	WhitelistedTokenSet  map[string]bool                 `mapstructure:"whitelistedTokenSet"`
	MaxGoroutines        int                             `mapstructure:"maxGoroutines"`
	CorrelatedPairConfig CorrelatedPairConfig            `mapstructure:"correlatedPair" json:"correlatedPair"`
}

type CorrelatedPairConfig struct {
	ChainName              string  `mapstructure:"chainName"`
	MinLiquidityScore      float64 `mapstructure:"minLiquidityScore"`
	MinLiquidityScoreLevel float64 `mapstructure:"minLiquidityScoreLevel"`
}
