package equalizer

import (
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexTypeEqualizer = "equalizer"

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

	bps = bignum.TenPowInt(18)
)
