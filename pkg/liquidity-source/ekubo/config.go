package ekubo

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPConfig struct {
	BaseURL    string                `json:"baseUrl,omitempty"`
	Timeout    durationjson.Duration `json:"timeout,omitempty"`
	RetryCount int                   `json:"retryCount,omitempty"`
}

type Config struct {
	DexId            string              `json:"dexId"`
	HTTPConfig       HTTPConfig          `json:"httpConfig"`
	ChainId          valueobject.ChainID `json:"chainId"`
	Core             common.Address      `json:"core"`
	Oracle           common.Address      `json:"oracle"`
	Twamm            common.Address      `json:"twamm"`
	BasicDataFetcher string              `json:"basicDataFetcher"`
	TwammDataFetcher string              `json:"twammDataFetcher"`
	Router           string              `json:"router"`

	supportedExtensions map[common.Address]ExtensionType
}

func (c *Config) SupportedExtensions() map[common.Address]ExtensionType {
	if c.supportedExtensions == nil {
		c.supportedExtensions = map[common.Address]ExtensionType{
			{}:       ExtensionTypeBase,
			c.Oracle: ExtensionTypeOracle,
			c.Twamm:  ExtensionTypeTwamm,
		}
	}
	return c.supportedExtensions
}
