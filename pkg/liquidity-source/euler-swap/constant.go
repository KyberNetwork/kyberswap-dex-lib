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

	vaultMethodCash          = "cash"
	vaultMethodDebtOf        = "debtOf"
	vaultMethodMaxDeposit    = "maxDeposit"
	vaultMethodCaps          = "caps"
	vaultMethodTotalBorrows  = "totalBorrows"
	vaultMethodTotalAssets   = "totalAssets"
	vaultMethodTotalSupply   = "totalSupply"
	vaultMethodBalanceOf     = "balanceOf"
	vaultMethodOracle        = "oracle"
	vaultMethodUnitOfAccount = "unitOfAccount"
	vaultMethodAsset         = "asset"
	vaultMethodDecimals      = "decimals"
	vaultMethodLTVBorrow     = "LTVBorrow"

	evcMethodIsAccountOperatorAuthorized = "isAccountOperatorAuthorized"
	evcMethodGetCollaterals              = "getCollaterals"
	evcMethodGetControllers              = "getControllers"
	evcMethodIsControllerEnabled         = "isControllerEnabled"

	routerMethodGetQuotes = "getQuotes"

	batchSize = 100
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidAmountIn   = errors.New("invalid amount in")
	ErrInvalidAmountOut  = errors.New("invalid amount out")
	ErrSwapIsPaused      = errors.New("swap is paused")
	ErrMultiDebts        = errors.New("multiple debts")
	ErrInsolvency        = errors.New("insolvency")
	ErrCurveViolation    = errors.New("curve violation")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrSwapLimitExceeded = errors.New("swap limit exceed")

	bufferSwapLimit = uint256.NewInt(90) // 90%
)
