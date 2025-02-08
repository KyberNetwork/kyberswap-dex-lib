package overnightusdp

import (
	"errors"
	"math/big"
)

const (
	DexType = "overnight-usdp"

	exchangeMethodPaused    = "paused"
	exchangeMethodBuyFee    = "buyFee"
	exchangeMethodRedeemFee = "redeemFee"

	exchangeMethodUsdc    = "usdc"
	exchangeMethodUsdPlus = "usdPlus"

	erc20MethodDecimals = "decimals"

	defaultReserves       = "1000000000000000000000000"
	defaultGas      int64 = 200000
)

var (
	buyFeeDenominator    = big.NewInt(100000)
	redeemFeeDenominator = big.NewInt(100000)
)

var (
	ErrPoolIsPaused       = errors.New("pool is paused")
	ErrorInvalidTokenIn   = errors.New("invalid tokenIn")
	ErrorInvalidAmountIn  = errors.New("AmountIn is zero")
	ErrorInvalidAmountOut = errors.New("AmountOut is zero")
)
