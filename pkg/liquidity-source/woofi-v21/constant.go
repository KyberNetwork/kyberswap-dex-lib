package woofiv21

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

const (
	DexTypeWooFiV21 = "woofi-v21"

	integrationHelperMethodAllBaseTokens = "allBaseTokens"

	wooPPV2MethodQuoteToken = "quoteToken"
	wooPPV2MethodTokenInfos = "tokenInfos"
	wooPPV2MethodWooracle   = "wooracle"
	wooPPV2MethodPaused     = "paused"

	wooracleMethodWoState       = "woState"
	wooracleMethodDecimals      = "decimals"
	wooracleMethodClOracles     = "clOracles"
	wooracleMethodTimestamp     = "timestamp"
	wooracleMethodBound         = "bound"
	wooracleMethodStaleDuration = "staleDuration"

	cloracleMethodLatestRoundData = "latestRoundData"

	erc20MethodDecimals = "decimals"
)

var (
	DefaultGas          = Gas{Swap: 300000}
	Number_1e5          = number.TenPow(5)
	safetyBufferPercent = uint256.NewInt(80) // 80%
)
