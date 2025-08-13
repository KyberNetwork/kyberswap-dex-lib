package eulerswap

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "euler-swap"

	DefaultGas int64 = 640910

	factoryMethodPoolsSlice  = "poolsSlice"
	factoryMethodPoolsLength = "poolsLength"

	poolMethodGetAssets   = "getAssets"
	poolMethodGetReserves = "getReserves"
	poolMethodGetParams   = "getParams"
	poolMethodEVC         = "EVC"

	vaultMethodCash         = "cash"
	vaultMethodDebtOf       = "debtOf"
	vaultMethodMaxDeposit   = "maxDeposit"
	vaultMethodCaps         = "caps"
	vaultMethodTotalBorrows = "totalBorrows"
	vaultMethodBalanceOf    = "balanceOf"

	vaultMethodTotalAssets      = "totalAssets"
	vaultMethodTotalSupply      = "totalSupply"
	vaultMethodAccountLiquidity = "accountLiquidity"
	vaultMethodUnitOfAccount    = "unitOfAccount"
	vaultMethodOracle           = "oracle"
	vaultMethodLTVBorrow        = "LTVBorrow"

	evcMethodIsAccountOperatorAuthorized = "isAccountOperatorAuthorized"

	routerMethodGetQuotes = "getQuotes"

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

	ConfigScale = uint256.NewInt(1e4)

	bufferSwapLimit = uint256.NewInt(85) // 85%
)
