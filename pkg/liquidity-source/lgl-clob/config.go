package lglclob

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}

type Config struct {
	DexID         string              `mapstructure:"dexID" json:"dexID,omitempty"`
	ChainId       valueobject.ChainID `mapstructure:"chain_id" json:"chain_id,omitempty"`
	HTTPConfig    HTTPConfig          `mapstructure:"http_config" json:"http_config,omitempty"`
	HelperAddress string              `mapstructure:"helper_address" json:"helper_address,omitempty"`
}
