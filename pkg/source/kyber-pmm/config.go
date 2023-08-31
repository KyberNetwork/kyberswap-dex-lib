package kyberpmm

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID       string            `json:"dexID"`
	HTTP        HTTPConfig        `mapstructure:"http"`
	MemoryCache MemoryCacheConfig `mapstructure:"memory_cache" json:"memory_cache"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url"`
	Timeout    durationjson.Duration `mapstructure:"timeout"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count"`
}

type MemoryCacheConfig struct {
	TTL struct {
		Tokens      durationjson.Duration `mapstructure:"tokens"`
		Pairs       durationjson.Duration `mapstructure:"pairs"`
		PriceLevels durationjson.Duration `mapstructure:"price_levels"`
	} `mapstructure:"ttl"`
}
