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
}

type UpdateLiquidityScoreConfig struct {
	MeanType            string                          `mapstructure:"meanType" json:"meanType"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
}
