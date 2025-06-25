package aavev3

import "errors"

const (
	DexType = "aave-v3"

	defaultGas = 150000

	// Aave V3 Pool methods
	poolMethodGetReserveData         = "getReserveData"
	poolMethodGetReserveDataExtended = "getReserveDataExtended"
	poolMethodGetReservesList        = "getReservesList"
	poolMethodGetReservesCount       = "getReservesCount"

	// Aave V3 AToken methods
	atokenMethodBalanceOf   = "balanceOf"
	atokenMethodTotalSupply = "totalSupply"

	// Aave V3 VariableDebtToken methods
	variableDebtTokenMethodBalanceOf   = "balanceOf"
	variableDebtTokenMethodTotalSupply = "totalSupply"

	// Aave V3 StableDebtToken methods
	stableDebtTokenMethodBalanceOf   = "balanceOf"
	stableDebtTokenMethodTotalSupply = "totalSupply"
)

var (
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrReserveNotActive         = errors.New("RESERVE_NOT_ACTIVE")
	ErrReserveFrozen            = errors.New("RESERVE_FROZEN")
	ErrReservePaused            = errors.New("RESERVE_PAUSED")
)
