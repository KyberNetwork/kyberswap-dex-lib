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
	Twamm                   common.Address       `json:"twamm"`
	MevCapture              common.Address       `json:"mevCapture"`
	BoostedFeesConcentrated common.Address       `json:"boostedFeesConcentrated"`
	MevCaptureRouter        common.Address       `json:"mevCaptureRouter"`
	QuoteDataFetcher        string               `json:"quoteDataFetcher"`
	TwammDataFetcher        string               `json:"twammDataFetcher"`
	BoostedFeesDataFetcher  string               `json:"boostedFeesDataFetcher"`

	supportedExtensions map[common.Address]ExtensionType
}

func NewConfig(
	dexId valueobject.Exchange,
	chainId valueobject.ChainID,
	subgraphAPI string,
	core, oracle, twamm, mevCapture, boostedFeesConcentrated, mevCaptureRouter common.Address,
	quoteDataFetcher, twammDataFetcher, boostedFeesDataFetcher string,
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
		MevCaptureRouter:        mevCaptureRouter,
		QuoteDataFetcher:        quoteDataFetcher,
		TwammDataFetcher:        twammDataFetcher,
		BoostedFeesDataFetcher:  boostedFeesDataFetcher,

		supportedExtensions: nil,
	}
}

func (c *Config) ExtensionType(extension common.Address) ExtensionType {
	if c.supportedExtensions == nil {
		c.supportedExtensions = map[common.Address]ExtensionType{
			c.Oracle:                  ExtensionTypeOracle,
			c.Twamm:                   ExtensionTypeTwamm,
			c.MevCapture:              ExtensionTypeMevCapture,
			c.BoostedFeesConcentrated: ExtensionTypeBoostedFeesConcentrated,
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
