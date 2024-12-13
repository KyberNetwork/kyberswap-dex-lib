package dexT1

import "math/big"

const (
	DexType = "fluid-dex-t1"
)

const (
	// DexReservesResolver methods
	DRRMethodGetAllPoolsReservesAdjusted = "getAllPoolsReservesAdjusted"
	DRRMethodGetPoolReservesAdjusted     = "getPoolReservesAdjusted"

	// ERC20 Token methods
	TokenMethodDecimals = "decimals"

	// StorageRead methods
	SRMethodReadFromStorage = "readFromStorage"
)

const (
	String1e18 = "1000000000000000000"
	String1e27 = "1000000000000000000000000000"

	DexAmountsDecimals = 12

	FeePercentPrecision    int64 = 1e4
	Fee100PercentPrecision int64 = 1e6

	MaxPriceDiff int64 = 5 // 5%

	MinSwapLiquidity int64 = 6667 // on-chain we use 1e4 but use extra buffer for potential price diff using pool price vs center price at the check
)

var bI1e18, _ = new(big.Int).SetString(String1e18, 10) // 1e18
var bI1e27, _ = new(big.Int).SetString(String1e27, 10) // 1e27
var bI10 = new(big.Int).SetInt64(10)
var bI100 = new(big.Int).SetInt64(100)
