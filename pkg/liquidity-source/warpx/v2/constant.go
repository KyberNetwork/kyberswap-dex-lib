package warpx

import (
	"errors"
)

const (
	DexType = "warpx"

	defaultGas = 76562

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodReserve0    = "reserve0"
	pairMethodReserve1    = "reserve1"
	pairMethodGetReserves = "getReserves"
)

var (
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                 = errors.New("K")
)
