package equalizer

import "math/big"

const (
	DexTypeEqualizer = "equalizer"

	defaultTokenWeight = 50

	poolFactoryMethodAllPairLength = "allPairsLength"
	poolFactoryMethodAllPairs      = "allPairs"
	poolFactoryMethodStableFee     = "stableFee"
	poolFactoryMethodVolatileFee   = "volatileFee"
	poolFactoryMethodGetRealFee    = "getRealFee"

	poolMethodMetadata    = "metadata"
	poolMethodGetReserves = "getReserves"

	reserveZero = "0"
)

var (
	DefaultGas = Gas{Swap: 227000}

	bps = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)
