package integral

import "math/big"

var (
	defaultGas = Gas{Swap: 227000}
	precison   = big.NewInt(1e18)

	FEES_BASE *big.Int = big.NewInt(1000000)

	// pair methods
	pairToken0Method = "token0"
	pairToken1Method = "token1"

	pairMintFeeMethod = "mintFee"
	pairBurnFeeMethod = "burnFee"
	pairSwapFeeMethod = "swapFee"

	pairOracleMethod = "oracle"

	factoryAllPairsMethod       = "allPairs"
	factoryAllPairsLengthMethod = "allPairsLength"

	libraryGetReservesMethod = "getReserves"
	libraryGetFeesMethod     = "getFees"
)
