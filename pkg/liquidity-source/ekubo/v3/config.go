package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId            valueobject.Exchange `json:"dexId"`
	ChainId          valueobject.ChainID  `json:"chainId"`
	SubgraphAPI      string               `json:"subgraphAPI"`
	Core             common.Address       `json:"core"`
	Oracle           common.Address       `json:"oracle"`
	Twamm            common.Address       `json:"twamm"`
	MevCapture       common.Address       `json:"mevCapture"`
	QuoteDataFetcher string               `json:"quoteDataFetcher"`
	TwammDataFetcher string               `json:"twammDataFetcher"`

	supportedExtensions map[common.Address]ExtensionType
}

func (c *Config) SupportedExtensions() map[common.Address]ExtensionType {
	if c.supportedExtensions == nil {
		c.supportedExtensions = map[common.Address]ExtensionType{
			{}:           ExtensionTypeBase,
			c.Oracle:     ExtensionTypeOracle,
			c.Twamm:      ExtensionTypeTwamm,
			c.MevCapture: ExtensionTypeMevCapture,
		}
	}

	return c.supportedExtensions
}
