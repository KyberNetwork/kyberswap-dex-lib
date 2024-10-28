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
)

const (
	String1e18 = "1000000000000000000"

	DexAmountsDecimals int64 = 12

	FeePercentPrecision    int64 = 1e4
	Fee100PercentPrecision int64 = 1e6
)

var bI1e18, _ = new(big.Int).SetString(String1e18, 10) // 1e18
