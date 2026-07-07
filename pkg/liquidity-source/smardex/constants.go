package smardex

import (
	"math/big"

	"github.com/holiman/uint256"
)

var (
	DefaultGas                 = Gas{Swap: 160000}
	FEES_BASE                  = big.NewInt(1000000)
	FEES_BASE_ETHEREUM         = big.NewInt(10000)
	MAX_BLOCK_DIFF_SECONDS     = uint256.NewInt(300)
	FEES_LP_DEFAULT_ETHEREUM   = big.NewInt(5)
	FEES_POOL_DEFAULT_ETHEREUM = big.NewInt(2)
)

const (
	DexTypeSmardex = "smardex"
	reserveZero    = "0"

	// factory methods
	factoryAllPairsLengthMethod = "allPairsLength"
	factoryAllPairsMethod       = "allPairs"

	// pair methods
	pairToken0Method             = "token0"
	pairToken1Method             = "token1"
	pairGetPairFeesMethod        = "getPairFees"
	pairGetFeeToAmountsMethod    = "getFeeToAmounts"
	pairGetFeesMethod            = "getFees"
	pairGetFictiveReservesMethod = "getFictiveReserves"
	pairGetPriceAverageMethod    = "getPriceAverage"
	pairGetReservesMethod        = "getReserves"
)
