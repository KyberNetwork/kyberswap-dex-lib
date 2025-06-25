package euler

import (
	"errors"
)

const (
	DexType = "uniswap-v4-euler"

	DefaultGas int64 = 400000

	factoryMethodPoolsSlice  = "poolsSlice"
	factoryMethodPoolsLength = "poolsLength"

	poolMethodGetAssets           = "getAssets"
	poolMethodGetReserves         = "getReserves"
	poolMethodGetParams           = "getParams"
	poolMethodEVC                 = "EVC"
	poolMethodEulerAccount        = "eulerAccount"
	poolMethodAsset0              = "asset0"
	poolMethodAsset1              = "asset1"
	poolMethodVault0              = "vault0"
	poolMethodVault1              = "vault1"
	poolMethodPriceX              = "priceX"
	poolMethodPriceY              = "priceY"
	poolMethodEquilibriumReserve0 = "equilibriumReserve0"
	poolMethodEquilibriumReserve1 = "equilibriumReserve1"
	poolMethodConcentrationX      = "concentrationX"
	poolMethodConcentrationY      = "concentrationY"
	poolMethodFeeMultiplier       = "feeMultiplier"

	vaultMethodCash            = "cash"
	vaultMethodDebtOf          = "debtOf"
	vaultMethodMaxDeposit      = "maxDeposit"
	vaultMethodCaps            = "caps"
	vaultMethodTotalBorrows    = "totalBorrows"
	vaultMethodBalanceOf       = "balanceOf"
	vaultMethodConvertToAssets = "convertToAssets"
	vaultMethodTotalAssets     = "totalAssets"
	vaultMethodTotalSupply     = "totalSupply"

	evcMethodIsAccountOperatorAuthorized = "isAccountOperatorAuthorized"

	batchSize = 100
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
	ErrInvalidAmountOut  = errors.New("invalid amount out")
	ErrSwapIsPaused      = errors.New("swap is paused")
	ErrOverflow          = errors.New("math overflow")
	ErrCurveViolation    = errors.New("curve violation")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrSwapLimitExceeded = errors.New("swap limit exceed")
)
