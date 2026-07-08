package stabull

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID          string     `json:"dexID"`
	ChainID        uint       `json:"chainID"`
	FactoryAddress string     `json:"factoryAddress"`
	NewPoolLimit   int        `json:"newPoolLimit"` // Batch size for pool discovery
	HTTPConfig     HTTPConfig `mapstructure:"http_config" json:"http_config,omitempty"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}
