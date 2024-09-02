package integral

import (
	"math/big"

	"github.com/holiman/uint256"
)

var (
	defaultGas = Gas{Swap: 227000}
	precison   = uint256.NewInt(1e18)

	FEES_BASE *big.Int = big.NewInt(1000000)

	// pair methods
	pairToken0Method = "token0"
	pairToken1Method = "token1"

	pairSwapFeeMethod = "swapFee"

	pairOracleMethod = "oracle"

	factoryAllPairsMethod       = "allPairs"
	factoryAllPairsLengthMethod = "allPairsLength"

	libraryGetReservesMethod = "getReserves"

	oracleDecimalsConverterMethod = "decimalsConverter"
)
