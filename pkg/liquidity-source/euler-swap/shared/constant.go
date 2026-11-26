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

// QuoteAmount is the inAmount of every oracle quote the trackers ask for, whatever the asset or the unit of
// account. A quote becomes a price per smallest unit (UnitPrice: quote * PriceScale / QuoteAmount), and Euler's
// oracle quotes are linear in inAmount, so the price does not depend on the amount quoted as long as it is large
// enough for the quote not to round away. The stored price is floor(PriceScale * price per smallest unit) once
// QuoteAmount >= PriceScale, so PriceScale bounds its precision and a larger amount buys none: QuoteAmount is
// PriceScale, 10^18. A fixed amount needs no decimals() read, which would cost a round trip and fail a batch
// when it fails. It stays below the uint128 maximum (~3.4e38) that Euler's UniswapV3 oracle adapter requires of
// inAmount, with 20 digits to spare for an adapter that feeds one leg's output into the next (a cross adapter),
// and it keeps the output of an oracle that scales by 1e18 well inside uint256. It is shared: do not mutate it.
var QuoteAmount = bignumber.TenPowInt(18)

var (
	BufferSwapLimit = uint256.NewInt(9990) // 99.90%
	VirtualAmount   = big.NewInt(1e6)

	E36        = big256.TenPow(36)
	MaxUint112 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(big256.U1, 112), 1) // 2^112 - 1
	E18Int     = int256.NewInt(1e18)                                                 // 1e18
	RA         = uint256.NewInt(3814697265625)
)
