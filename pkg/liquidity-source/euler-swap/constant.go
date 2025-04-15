package eulerswap

import (
	"github.com/holiman/uint256"
)

const (
	DexType = "euler-swap"
)

var (
	defaultGas = Gas{Swap: 225000}

	oneE18         = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18)) // 1e18
	VIRTUAL_AMOUNT = uint256.NewInt(1e6)                                          // 1e6
)

const (
	factoryMethodPoolsLength = "poolsLength"
	factoryMethodPools       = "allPools"

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
