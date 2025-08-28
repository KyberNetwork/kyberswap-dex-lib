package velodromev1

import (
	"errors"
)

const (
	DexType       = "velodrome"
	DexTypeRamses = "ramses"

	pairFactoryMethodIsPaused       = "isPaused"
	pairFactoryMethodAllPairs       = "allPairs"
	pairFactoryMethodAllPairsLength = "allPairsLength"
	pairMethodMetadata              = "metadata"
	pairMethodGetReserves           = "getReserves"

	genericMethodFee        = "fee"
	genericTemplatePool     = "pool"
	genericTemplateFactory  = "factory"
	genericTemplateIsStable = "isStable"

	defaultGas = 227000
)

var (
	ErrPoolIsPaused             = errors.New("pool is paused")
	ErrInvalidAmountIn          = errors.New("invalid amountIn")
	ErrInvalidAmountOut         = errors.New("invalid amountOut")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrK                        = errors.New("K")
	ErrUnimplemented            = errors.New("unimplemented")
)
