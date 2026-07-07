package ringswap

import (
	"errors"
)

const (
	DexType    = "ringswap"
	defaultGas = 225000
)

const (
	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
)

const (
	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"

	pairMethodBalanceOf = "balanceOf"

	fewWrappedTokenGetTokenMethod = "token"
)

var (
	ErrReserveIndexOutOfBounds = errors.New("reserve index out of bounds")
	ErrTokenIndexOutOfBounds   = errors.New("token index out of bounds")
	ErrTokenSwapNotAllowed     = errors.New("cannot swap between original token and wrapped token")

	ErrNoSwapLimit = errors.New("swap limit is required")
)
