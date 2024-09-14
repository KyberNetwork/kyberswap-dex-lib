package integral

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	DexTypeIntegral = "integral"

	defaultGas = Gas{Swap: 227000}
	precison   = uint256.NewInt(1e18)

	// errors
	ErrTokenNotFound  = errors.New("tokens not found")
	ErrInvalidTokenIn = errors.New("invalid tokenIn")
	ErrTP2E           = errors.New("TP2E")
	ErrTP07           = errors.New("TP07")
	ErrTP08           = errors.New("TP08")
	ErrTP31           = errors.New("TP31")
	ErrTP02           = errors.New("TP02")
	ErrT027           = errors.New("T027")
	ErrT028           = errors.New("T028")
	ErrSM43           = errors.New("SM43")
	ErrSM4E           = errors.New("SM4E")
	ErrSM12           = errors.New("SM12")
	ErrSM2A           = errors.New("SM2A")
	ErrSM4D           = errors.New("SM4D")
	ErrSM11           = errors.New("SM11")
	ErrSM29           = errors.New("SM29")
	ErrSM42           = errors.New("SM42")

	ErrTD37 = errors.New("TD37")

	// pair methods
	pairToken0Method  = "token0"
	pairToken1Method  = "token1"
	pairSwapFeeMethod = "swapFee"
	pairOracleMethod  = "oracle"

	// factory methods
	factoryAllPairsMethod       = "allPairs"
	factoryAllPairsLengthMethod = "allPairsLength"

	// relayer methods
	relayerFactoryMethod          = "factory"
	relayerGetTokenLimitMinMethod = "getTokenLimitMin"

	// reserves methods
	libraryGetReservesMethod = "getReserves"

	// oracle methods
	oracleGetPriceInfoMethod    = "getPriceInfo"
	oracleGetAveragePriceMethod = "getAveragePrice"
	oracleGetSpotPriceMethod    = "getSpotPrice"
	oracleXDecimalsMethod       = "xDecimals"
	oracleYDecimalsMethod       = "yDecimals"
)
