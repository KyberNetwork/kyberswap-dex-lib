package virtualfun

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	defaultGas = Gas{Swap: 250000}

	bondingCurveApplicationGas int64 = 5_000_000

	ZERO_ADDRESS = common.Address{}

	U100 = uint256.NewInt(100)
	ZERO = uint256.NewInt(0)
)

const (
	DexType = "virtual-fun"

	erc20BalanceOfMethod = "balanceOf"

	pairTokenAMethod      = "tokenA"
	pairTokenBMethod      = "tokenB"
	pairGetReservesMethod = "getReserves"
	pairKLastMethod       = "kLast"

	factoryAllPairsLengthMethod = "allPairsLength"
	factoryGetPairMethod        = "pairs"

	factorySellTaxMethod = "sellTax"
	factoryBuyTaxMethod  = "buyTax"

	bondingUnwrapTokenMethod   = "unwrapToken"
	bondingGradThresholdMethod = "gradThreshold"
)
