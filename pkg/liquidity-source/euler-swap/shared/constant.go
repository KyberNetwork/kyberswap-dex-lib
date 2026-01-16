package shared

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	PoolMethodGetAssets        = "getAssets"
	PoolMethodGetReserves      = "getReserves"
	PoolMethodGetParams        = "getParams"        // V1
	PoolMethodGetStaticParams  = "getStaticParams"  // V2
	PoolMethodGetDynamicParams = "getDynamicParams" // V2
	PoolMethodIsInstalled      = "isInstalled"      // V2
	PoolMethodEVC              = "EVC"

	FactoryMethodPoolsLength = "poolsLength"
	FactoryMethodPoolsSlice  = "poolsSlice"

	VaultMethodCash            = "cash"
	VaultMethodDebtOf          = "debtOf"
	VaultMethodMaxDeposit      = "maxDeposit"
	VaultMethodCaps            = "caps"
	VaultMethodTotalBorrows    = "totalBorrows"
	VaultMethodTotalAssets     = "totalAssets"
	VaultMethodTotalSupply     = "totalSupply"
	VaultMethodBalanceOf       = "balanceOf"
	VaultMethodConvertToAssets = "convertToAssets"
	VaultMethodOracle          = "oracle"
	VaultMethodUnitOfAccount   = "unitOfAccount"
	VaultMethodAsset           = "asset"
	VaultMethodDecimals        = "decimals"
	VaultMethodLTVBorrow       = "LTVBorrow"

	EvcMethodIsAccountOperatorAuthorized = "isAccountOperatorAuthorized"
	EvcMethodGetCollaterals              = "getCollaterals"
	EvcMethodGetControllers              = "getControllers"
	EvcMethodIsControllerEnabled         = "isControllerEnabled"

	RouterMethodGetQuotes = "getQuotes"

	BatchSize = 100
)

var (
	BufferSwapLimit = uint256.NewInt(9990) // 99.90%
	VirtualAmount   = big.NewInt(1e6)

	E36        = big256.TenPow(36)
	MaxUint112 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(big256.U1, 112), 1) // 2^112 - 1
	E18Int     = int256.NewInt(1e18)                                                 // 1e18
	RA         = uint256.NewInt(3814697265625)
)
