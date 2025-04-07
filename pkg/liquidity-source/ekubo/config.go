package ekubo

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPConfig struct {
	BaseURL    string                `json:"base_url,omitempty"`
	Timeout    durationjson.Duration `json:"timeout,omitempty"`
	RetryCount int                   `json:"retry_count,omitempty"`
}

type Config struct {
	DexID       string                   `json:"dexID"`
	ChainID     valueobject.ChainID      `json:"chainID"`
	Core        string                   `json:"core"`
	Oracle      string                   `json:"oracle"`
	DataFetcher string                   `json:"data_fetcher"`
	Router      string                   `json:"router"`
	HTTPConfig  HTTPConfig               `json:"http_config"`
	Extensions  map[string]ExtensionType `json:"extensions"`
}
