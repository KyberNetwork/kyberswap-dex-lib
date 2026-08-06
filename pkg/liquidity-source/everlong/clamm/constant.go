package everlongclamm

import (
	"errors"
)

const (
	DexType = "everlong-clamm"

	poolManagerMethodGetSlot0     = "getSlot0"
	poolManagerMethodGetLiquidity = "getLiquidity"
	almMethodGetRungs             = "getRungs"
	almMethodPoolFee              = "poolFee"
)

var (
	ErrExactOutNotSupported = errors.New("exact-output swaps are not supported (hook reverts ExactInputOnly)")
	ErrInvalidToken         = errors.New("invalid token")
	ErrZeroAmountOut        = errors.New("zero amount out")
	ErrPoolNotInitialized   = errors.New("pool is not initialized on the pool manager")
	ErrInvalidRungs         = errors.New("invalid rung ladder from ALM.getRungs")
)
