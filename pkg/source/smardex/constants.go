package smardex

import "math/big"

var (
	DefaultGas                      = Gas{Swap: 160000}
	FEES_BASE              *big.Int = big.NewInt(1000000)
	MAX_BLOCK_DIFF_SECONDS *big.Int = big.NewInt(300)
	FEES_LP_DEFAULT        *big.Int = big.NewInt(700)
	FEES_POOL_DEFAULT      *big.Int = big.NewInt(300)
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
	pairTotalSupplyMethod        = "totalSupply"
)
