package camelot

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const DexTypeCamelot = "camelot"

const (
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodFeeTo          = "feeTo"
	factoryMethodOwnerFeeShare  = "ownerFeeShare"

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

var (
	DefaultGas = Gas{Swap: 128000}

	ZeroAddress = common.Address{}

	ErrInsufficientOutputAmount = errors.New("CamelotPair: INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("CamelotPair: INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                 = errors.New("CamelotPair: K")
)
