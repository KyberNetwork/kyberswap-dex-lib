package eulerswap

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "euler-swap"
)

var (
	defaultGas = Gas{Swap: 400000}

	VIRTUAL_AMOUNT = uint256.NewInt(1e6) // 1e6

	ErrInvalidVaults     = errors.New("invalid vaults")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInvalidReserve    = errors.New("invalid reserve")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrSwapIsPaused      = errors.New("swap is paused")
	ErrOverflow          = errors.New("math overflow")
	ErrCurveViolation    = errors.New("curve violation")
	ErrSwapLimitExceeded = errors.New("swap limit exceed")
)

const (
	factoryMethodPoolsLength = "poolsLength"
	factoryMethodPoolsSlice  = "poolsSlice"

	poolMethodEulerAccount        = "eulerAccount"
	poolMethodAsset0              = "asset0"
	poolMethodAsset1              = "asset1"
	poolMethodGetReserves         = "getReserves"
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
)
