package kyberpmm

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID              string            `json:"dexID,omitempty"`
	RFQContractAddress string            `mapstructure:"rfq_contract_address" json:"rfq_contract_address,omitempty"`
	HTTP               HTTPConfig        `mapstructure:"http" json:"http,omitempty"`
	MemoryCache        MemoryCacheConfig `mapstructure:"memory_cache" json:"memory_cache,omitempty"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}

type MemoryCacheConfig struct {
	TTL struct {
		Tokens      durationjson.Duration `mapstructure:"tokens" json:"tokens,omitempty"`
		Pairs       durationjson.Duration `mapstructure:"pairs" json:"pairs,omitempty"`
		PriceLevels durationjson.Duration `mapstructure:"price_levels" json:"price_levels,omitempty"`
	} `mapstructure:"ttl"`
}
