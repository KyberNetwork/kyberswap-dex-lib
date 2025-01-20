package liquiditybookv21

import (
	"math/big"
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
	pairGetPriceFromID                 = "getPriceFromId"
)

const (
	defaultTokenWeight = 50

	graphFirstLimit = 1000

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/Constants.sol#L20
	basisPointMax = 10000

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/Constants.sol#L11
	scaleOffset = 128

	// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/PriceHelper.sol#L20
	realIDShift = 1 << 23

	defaultGas = 125000
)

var (
	scale     = new(big.Int).Lsh(big.NewInt(1), scaleOffset)
	precision = big.NewInt(1e18)

	maxFee = big.NewInt(1e17)
	u, _   = new(big.Int).SetString("100000", 16)
)
