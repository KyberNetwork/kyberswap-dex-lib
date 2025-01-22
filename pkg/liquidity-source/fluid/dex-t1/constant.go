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

	// TokenMethodDecimals - ERC20 Token methods
	TokenMethodDecimals = "decimals"

	// SRMethodReadFromStorage - StorageRead methods
	SRMethodReadFromStorage = "readFromStorage"
)

const (
	DexAmountsDecimals = 12

	FeePercentPrecision float64 = 1e4
)

var (
	MaxPriceDiff           = big.NewInt(5)      // 5%
	MinSwapLiquidity       = big.NewInt(0.85e4) // on-chain we use 1e4 but use extra buffer to avoid reverts
	Fee100PercentPrecision = big.NewInt(1e6)

	bI100  = big.NewInt(100)
	bI1e18 = bignumber.TenPowInt(18)
	bI1e27 = bignumber.TenPowInt(27)
)
