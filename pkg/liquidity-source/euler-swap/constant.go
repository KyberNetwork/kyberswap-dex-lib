package eulerswap

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType     = "euler-swap"
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

var (
	defaultGas = Gas{Swap: 225000}

	oneE18 = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18)) // 1e18
)

const (
	factoryMethodAllPoolsLength = "allPoolsLength"
	factoryMethodAllPools       = "allPools"
)

const (
	poolMethodEulerAccount = "eulerAccount"

	poolMethodPause       = "pause"
	poolMethodAsset0      = "asset0"
	poolMethodAsset1      = "asset1"
	poolMethodGetReserves = "getReserves"
	poolMethodVault0      = "vault0"
	poolMethodVault1      = "vault1"

	poolMethodPriceX = "priceX"
	poolMethodPriceY = "priceY"

	poolMethodEquilibriumReserve0 = "equilibriumReserve0"
	poolMethodEquilibriumReserve1 = "equilibriumReserve1"

	poolMethodConcentrationX = "concentrationX"
	poolMethodConcentrationY = "concentrationY"

	poolMethodFeeMultiplier = "feeMultiplier"

	vaultMethodCash            = "cash"
	vaultMethodDebtOf          = "debtOf"
	vaultMethodMaxDeposit      = "maxDeposit"
	vaultMethodMaxWithdraw     = "maxWithdraw"
	vaultMethodTotalBorrows    = "totalBorrows"
	vaultMethodBalanceOf       = "balanceOf"
	vaultMethodConvertToAssets = "convertToAssets"
)

var (
	ErrReserveIndexOutOfBounds = errors.New("reserve index out of bounds")
	ErrTokenIndexOutOfBounds   = errors.New("token index out of bounds")
	ErrTokenSwapNotAllowed     = errors.New("cannot swap between original token and wrapped token")

	ErrNoSwapLimit = errors.New("swap limit is required")
)
