package zkswapfinance

import "math/big"

const (
	DexTypeZkSwapFinance = "zkswap-finance"

	defaultTokenWeight = 50
	defaultSwapFee     = 30

	reserveZero = "0"
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	pairMethodToken0         = "token0"
	pairMethodToken1         = "token1"
	getReservesAndParameters = "getReservesAndParameters"
)

var (
	defaultGas = int64(60000)

	feePrecision = new(big.Int).SetUint64(10000) // fixed in contract
)
