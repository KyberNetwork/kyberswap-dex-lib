package ekubov3

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexId   string              `json:"dexId"`
	ChainId valueobject.ChainID `json:"chainId"`

	SubgraphAPI string `json:"subgraphAPI"`

	Core             common.Address `json:"core"`
	Oracle           common.Address `json:"oracle"`
	Twamm            common.Address `json:"twamm"`
	MevCapture       common.Address `json:"mevCapture"`
	QuoteDataFetcher string         `json:"quoteDataFetcher"`
	TwammDataFetcher string         `json:"twammDataFetcher"`

	SupportedExtensions map[common.Address]ExtensionType `json:"supportedExtensions"`
}

func NewConfig(chainId valueobject.ChainID, subgraphAPI string, core, oracle, twamm, mevCapture common.Address, quoteDataFetcher, twammDataFetcher string) *Config {
	return &Config{DexId: DexType, ChainId: chainId, SubgraphAPI: subgraphAPI, Core: core, Oracle: oracle, Twamm: twamm, MevCapture: mevCapture, QuoteDataFetcher: quoteDataFetcher, TwammDataFetcher: twammDataFetcher, SupportedExtensions: map[common.Address]ExtensionType{
		{}:         ExtensionTypeBase,
		oracle:     ExtensionTypeOracle,
		twamm:      ExtensionTypeTwamm,
		mevCapture: ExtensionTypeMevCapture,
	}}
}
