package woofiv2

import "github.com/ethereum/go-ethereum/common"

const (
	DexTypeWooFiV2 = "woofi-v2"

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

	zeroString = "0"
)

var (
	DefaultGas  = Gas{Swap: 125000}
	zeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
)
