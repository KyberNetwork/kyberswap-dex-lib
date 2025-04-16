package ekubo

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPConfig struct {
	BaseURL    string                `json:"baseUrl,omitempty"`
	Timeout    durationjson.Duration `json:"timeout,omitempty"`
	RetryCount int                   `json:"retryCount,omitempty"`
}

type Config struct {
	DexId       string                        `json:"dexId"`
	HTTPConfig  HTTPConfig                    `json:"httpConfig"`
	ChainId     valueobject.ChainID           `json:"chainId"`
	Core        string                        `json:"core"`
	DataFetcher string                        `json:"dataFetcher"`
	Router      string                        `json:"router"`
	Extensions  map[string]pool.ExtensionType `json:"extensions"`
}
