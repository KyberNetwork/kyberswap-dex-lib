package camelot

import "errors"

const DexTypeCamelot = "camelot"

const (
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodFeeTo          = "feeTo"
	factoryMethodOwnerFeeShare  = "ownerFeeShare"
)

const (
	pairMethodToken0               = "token0"
	pairMethodToken1               = "token1"
	pairMethodFeeDenominator       = "FEE_DENOMINATOR"
	pairMethodStableSwap           = "stableSwap"
	pairMethodToken0FeePercent     = "token0FeePercent"
	pairMethodToken1FeePercent     = "token1FeePercent"
	pairMethodPrecisionMultiplier0 = "precisionMultiplier0"
	pairMethodPrecisionMultiplier1 = "precisionMultiplier1"
	pairMethodGetReserves          = "getReserves"
)

const (
	defaultTokenWeight = 50
)

var (
	DefaultGas = Gas{Swap: 128000}

	ErrInsufficientOutputAmount = errors.New("CamelotPair: INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("CamelotPair: INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                 = errors.New("CamelotPair: K")
)
