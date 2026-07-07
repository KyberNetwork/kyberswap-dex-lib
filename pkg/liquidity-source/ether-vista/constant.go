package ethervista

import (
	"errors"
)

const (
	DexType = "ether-vista"
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodRouter         = "router"
)

const (
	pairMethodToken0       = "token0"
	pairMethodToken1       = "token1"
	pairMethodGetReserves  = "getReserves"
	pairMethodBuyTotalFee  = "buyTotalFee"
	pairMethodSellTotalFee = "sellTotalFee"
)

const (
	routerMethodUSDCToEth = "usdcToEth"
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
