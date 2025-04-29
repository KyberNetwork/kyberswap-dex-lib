package virtualfun

import (
	"errors"

	"github.com/holiman/uint256"
)

var (
	defaultGas = Gas{Swap: 250000}

	bondingCurveApplicationGas int64 = 5_000_000

	U100 = uint256.NewInt(100)
)

const (
	DexType = "virtual-fun"

	erc20BalanceOfMethod = "balanceOf"

	pairTokenAMethod      = "tokenA"
	pairTokenBMethod      = "tokenB"
	pairGetReservesMethod = "getReserves"
	pairKLastMethod       = "kLast"

	factoryAllPairsLengthMethod = "allPairsLength"
	factoryGetPairMethod        = "pairs"
	factorySellTaxMethod        = "sellTax"
	factoryBuyTaxMethod         = "buyTax"

	bondingUnwrapTokenMethod   = "unwrapToken"
	bondingGradThresholdMethod = "gradThreshold"
)

var (
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
)
