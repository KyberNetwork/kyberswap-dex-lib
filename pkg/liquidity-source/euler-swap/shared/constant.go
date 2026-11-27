package shared

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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
	VaultMethodLTVBorrow       = "LTVBorrow"

	EvcMethodIsAccountOperatorAuthorized = "isAccountOperatorAuthorized"
	EvcMethodGetCollaterals              = "getCollaterals"
	EvcMethodGetControllers              = "getControllers"
	EvcMethodIsControllerEnabled         = "isControllerEnabled"

	RouterMethodGetQuotes = "getQuotes"

	BatchSize = 100

	// BackupTailWindow bounds how many of the most-recently-appended pools
	// PoolsListUpdater rechecks each cycle. It's a backup for the primary
	// PoolFactory block-subscription flow, catching pools missed if a block was
	// skipped - not a full historical scan, so it only needs to look at the tail.
	BackupTailWindow = 5
)

// QuoteAmount is the inAmount of every oracle quote: 10^18 smallest units, needing no decimals() read. Quotes are
// linear in inAmount, so the quote is the price per smallest unit times 10^18, floored in the unit of account's
// decimals. It stays below the uint128 inAmount bound of Euler's UniswapV3 adapter. Do not mutate it.
var QuoteAmount = bignumber.TenPowInt(18)

var (
	BufferSwapLimit = uint256.NewInt(9990) // 99.90%
	VirtualAmount   = big.NewInt(1e6)

	E36        = big256.TenPow(36)
	MaxUint112 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(big256.U1, 112), 1) // 2^112 - 1
	E18Int     = int256.NewInt(1e18)                                                 // 1e18
	RA         = uint256.NewInt(3814697265625)
)
