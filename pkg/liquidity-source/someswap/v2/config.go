package someswapv2

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId      string
	ChainId    valueobject.ChainID
	Factory    string     `json:"factory,omitempty"`
	HTTPConfig HTTPConfig `mapstructure:"http_config" json:"http_config"`
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}
