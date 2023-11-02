package smardex

import "math/big"

var (
	DefaultGas                          = Gas{Swap: 160000}
	FEES_BASE                  *big.Int = big.NewInt(1000000)
	FEES_BASE_ETHEREUM         *big.Int = big.NewInt(10000)
	MAX_BLOCK_DIFF_SECONDS     *big.Int = big.NewInt(300)
	FEES_LP_DEFAULT_ETHEREUM   *big.Int = big.NewInt(5)
	FEES_POOL_DEFAULT_ETHEREUM *big.Int = big.NewInt(2)
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
