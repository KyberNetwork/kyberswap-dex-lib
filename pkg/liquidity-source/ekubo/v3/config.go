package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId                   valueobject.Exchange `json:"dexId"`
	ChainId                 valueobject.ChainID  `json:"chainId"`
	SubgraphAPI             string               `json:"subgraphAPI"`
	Core                    common.Address       `json:"core"`
	Oracle                  common.Address       `json:"oracle"`
	Twamm                   TwammConfig          `json:"twamm"`
	MevCapture              common.Address       `json:"mevCapture"`
	BoostedFeesConcentrated common.Address       `json:"boostedFeesConcentrated"`
	QuoteDataFetcher        string               `json:"quoteDataFetcher"`
	BoostedFeesDataFetcher  string               `json:"boostedFeesDataFetcher"`
	Ve33                    common.Address       `json:"ve33"`
	Ve33DataFetcher         string               `json:"ve33DataFetcher"`

	supportedExtensions map[common.Address]ExtensionType
}

type TwammConfig struct {
	V1 TwammDeployment `json:"v1"`
	V2 TwammDeployment `json:"v2"`
}

type TwammDeployment struct {
	Address     common.Address `json:"address"`
	DataFetcher string         `json:"dataFetcher"`
}

func NewConfig(
	dexId valueobject.Exchange,
	chainId valueobject.ChainID,
	subgraphAPI string,
	core, oracle common.Address,
	twamm TwammConfig,
	mevCapture, boostedFeesConcentrated common.Address,
	quoteDataFetcher, boostedFeesDataFetcher string,
) *Config {
	return &Config{
		DexId:                   dexId,
		ChainId:                 chainId,
		SubgraphAPI:             subgraphAPI,
		Core:                    core,
		Oracle:                  oracle,
		Twamm:                   twamm,
		MevCapture:              mevCapture,
		BoostedFeesConcentrated: boostedFeesConcentrated,
		QuoteDataFetcher:        quoteDataFetcher,
		BoostedFeesDataFetcher:  boostedFeesDataFetcher,

		supportedExtensions: nil,
	}
}

func (c *Config) TwammDataFetcher(address common.Address) string {
	switch address {
	case c.Twamm.V1.Address:
		return c.Twamm.V1.DataFetcher
	case c.Twamm.V2.Address:
		return c.Twamm.V2.DataFetcher
	default:
		return ""
	}
}

func (c *Config) ExtensionType(extension common.Address) ExtensionType {
	if c.supportedExtensions == nil {
		c.supportedExtensions = map[common.Address]ExtensionType{
			c.Oracle:                  ExtensionTypeOracle,
			c.Twamm.V1.Address:        ExtensionTypeTwamm,
			c.Twamm.V2.Address:        ExtensionTypeTwamm,
			c.MevCapture:              ExtensionTypeMevCapture,
			c.BoostedFeesConcentrated: ExtensionTypeBoostedFeesConcentrated,
		}
		if c.Ve33 != (common.Address{}) {
			c.supportedExtensions[c.Ve33] = ExtensionTypeVe33
		}
	}

	if extensionType, ok := c.supportedExtensions[extension]; ok {
		return extensionType
	}

	// Call points are encoded in the first byte of the extension address
	hasNoSwapCallPoints := extension.Bytes()[0]&0b01100000 == 0
	if hasNoSwapCallPoints {
		return ExtensionTypeNoSwapCallPoints
	}

	return ExtensionTypeUnknown
}
