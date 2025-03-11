package liquiditybookv21

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	scale          = new(uint256.Int).Lsh(big256.One, scaleOffset)
	uBasisPointMax = uint256.NewInt(basisPointMax)
	precision      = big256.BONE

	maxFee = uint256.NewInt(1e17)
	powU   = uint256.NewInt(0x100000)
)
