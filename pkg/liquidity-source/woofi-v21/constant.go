package woofiv21

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	DexTypeWooFiV21 = "woofi-v21"

	integrationHelperMethodAllBaseTokens = "allBaseTokens"

	wooPPV2MethodQuoteToken = "quoteToken"
	wooPPV2MethodTokenInfos = "tokenInfos"
	wooPPV2MethodWooracle   = "wooracle"

	wooracleMethodWoState       = "woState"
	wooracleMethodDecimals      = "decimals"
	wooracleMethodClOracles     = "clOracles"
	wooracleMethodTimestamp     = "timestamp"
	wooracleMethodBound         = "bound"
	wooracleMethodStaleDuration = "staleDuration"

	cloracleMethodLatestRoundData = "latestRoundData"

	erc20MethodDecimals = "decimals"

	defaultWeight = 1
	zeroString    = "0"
)

var (
	DefaultGas  = Gas{Swap: 300000}
	zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
)
