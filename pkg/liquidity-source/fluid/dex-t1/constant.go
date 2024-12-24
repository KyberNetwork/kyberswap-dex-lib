package dexT1

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "fluid-dex-t1"
)

const ( // DexReservesResolver methods
	DRRMethodGetAllPoolsReservesAdjusted = "getAllPoolsReservesAdjusted"
	DRRMethodGetPoolReservesAdjusted     = "getPoolReservesAdjusted"

	// TokenMethodDecimals for ERC20 Token methods
	TokenMethodDecimals = "decimals"

	// SRMethodReadFromStorage for StorageRead methods
	SRMethodReadFromStorage = "readFromStorage"
)

const (
	DexAmountsDecimals = 12

	FeePercentPrecision    int64 = 1e4
	Fee100PercentPrecision int64 = 1e6

	MaxPriceDiff int64 = 5 // 5%

	MinSwapLiquidity int64 = 6667 // on-chain we use 1e4 but use extra buffer for potential price diff using pool price vs center price at the check
)

var (
	bI10   = big.NewInt(10)
	bI100  = big.NewInt(100)
	bI1e18 = bignumber.NewBig10("1000000000000000000")          // 1e18
	bI1e27 = bignumber.NewBig10("1000000000000000000000000000") // 1e27
)
