package ekubov3

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	QuoteDataFetcherAddressStr = "0x5a3f0f1da4ac0c4b937d5685f330704c8e8303f1"
	TwammDataFetcherAddressStr = "0xc07e5b80750247c8b5d7234a9c79dfc58785392b"

	CoreAddressStr  = "0x00000000000014aA86C5d3c41765bb24e11bd701"
	TwammAddressStr = "0xd4F1060cB9c1A13e1d2d20379b8aa2cF7541eD9b"

	CoreAddressStrLower  = strings.ToLower(CoreAddressStr)
	TwammAddressStrLower = strings.ToLower(TwammAddressStr)

	CoreAddress       = common.HexToAddress(CoreAddressStr)
	OracleAddress     = common.HexToAddress("0x517E506700271AEa091b02f42756F5E174Af5230")
	TwammAddress      = common.HexToAddress(TwammAddressStr)
	MevCaptureAddress = common.HexToAddress("0x5555fF9Ff2757500BF4EE020DcfD0210CFfa41Be")

	SupportedExtensions = map[common.Address]ExtensionType{
		{}:                ExtensionTypeBase,
		OracleAddress:     ExtensionTypeOracle,
		TwammAddress:      ExtensionTypeTwamm,
		MevCaptureAddress: ExtensionTypeMevCapture,
	}
)

type Config struct {
	DexId   valueobject.Exchange `json:"dexId"`
	ChainId valueobject.ChainID  `json:"chainId"`
}

func NewConfig(chainId valueobject.ChainID) *Config {
	return &Config{ChainId: chainId}
}
