package woofiv2

const (
	DexTypeWooFiV2 = "woofi-v2"

	integrationHelperMethodAllBaseTokens = "allBaseTokens"

	wooPPV2MethodPaused     = "paused"
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
	DefaultGas = Gas{Swap: 125000}
)
