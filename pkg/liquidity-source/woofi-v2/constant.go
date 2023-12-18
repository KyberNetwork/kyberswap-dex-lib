package woofiv2

const (
	DexTypeWooFiV2 = "woofi-v2"

	integrationHelperMethodAllBaseTokens = "allBaseTokens"

	wooPPV2MethodQuoteToken = "quoteToken"
	wooPPV2MethodTokenInfos = "tokenInfos"
	wooPPV2MethodWooracle   = "wooracle"

	wooracleMethodWoState       = "woState"
	wooracleMethodDecimals      = "decimals"
	wooracleMethodTimestamp     = "timestamp"
	wooracleMethodBound         = "bound"
	wooracleMethodStaleDuration = "staleDuration"

	erc20MethodDecimals = "decimals"

	defaultWeight = 1
	zeroString    = "0"
)

var (
	DefaultGas = Gas{Swap: 125000}
)
