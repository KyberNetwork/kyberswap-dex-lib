package liquiditybookv21

import (
	"math/big"
	"time"
)

const (
	DexTypeLiquidityBookV21 = "liquiditybook-v21"
)

const (
	factoryMethodGetNumberOfLBPairs = "getNumberOfLBPairs"
	factoryMethodGetLBPairAtIndex   = "getLBPairAtIndex"

	pairMethodGetTokenX                = "getTokenX"
	pairMethodGetTokenY                = "getTokenY"
	pairMethodGetStaticFeeParameters   = "getStaticFeeParameters"
	pairMethodGetVariableFeeParameters = "getVariableFeeParameters"
	pairMethodGetReserves              = "getReserves"
	pairMethodGetBinStep               = "getBinStep"
	pairMethodGetActiveID              = "getActiveId"
)

const (
	defaultTokenWeight = 50

	graphQLRequestTimeout = 20 * time.Second
	graphFirstLimit       = 1000
	graphSkipLimit        = 5000

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/Constants.sol#L20
	basisPointMax = 10000

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/Constants.sol#L11
	scaleOffset = 128

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/PriceHelper.sol#L20
	realIDShift = 1 << 23
)

var (
	scale    = new(big.Int).Lsh(big.NewInt(1), scaleOffset)
	precison = new(big.Int).Lsh(big.NewInt(1), 18)
)
