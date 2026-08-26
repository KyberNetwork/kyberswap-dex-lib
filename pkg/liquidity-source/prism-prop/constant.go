package prismprop

import (
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangePrismProp

	methodGetSupportedPairs = "getSupportedPairs"
	methodGetOrderBook      = "getOrderBook"
	methodGetAmountOut      = "getAmountOut"

	// maxCalibratedFee bounds the self-calibrated fee (see pool_tracker.go's
	// calibrateFee) against a bad/zero reference quote producing a nonsense
	// fee that would make CalcAmountOut wildly under-quote.
	maxCalibratedFee = 0.01
)

// defaultGas is a placeholder pending a real gas measurement (e.g. via
// Tenderly simulation of an actual swap) -- see titan-prop/constant.go for
// the pattern this should follow once available.
var defaultGas = orderbook.Gas{Base: 200_000}
