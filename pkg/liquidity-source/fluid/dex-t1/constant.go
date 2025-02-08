package dexT1

import (
	"errors"
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
	MaxPriceDiff     = big.NewInt(5)      // 5%
	MinSwapLiquidity = big.NewInt(0.85e4) // on-chain we use 1e4 but use extra buffer to avoid reverts

	SIX_DECIMALS = big.NewInt(1e6)
	TWO_DECIMALS = big.NewInt(1e2)

	bI1e18 = bignumber.TenPowInt(18)
	bI1e27 = bignumber.TenPowInt(27)
)

var (
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")

	ErrInsufficientReserve    = errors.New("insufficient reserve: tokenOut amount exceeds reserve")
	ErrSwapAndArbitragePaused = errors.New("51043")

	ErrInsufficientWithdrawable = errors.New("insufficient reserve: tokenOut amount exceeds withdrawable limit")
	ErrInsufficientBorrowable   = errors.New("insufficient reserve: tokenOut amount exceeds borrowable limit")

	ErrInsufficientMaxPrice = errors.New("insufficient reserve: tokenOut amount exceeds max price limit")

	ErrVerifyReservesRatiosInvalid = errors.New("invalid reserves ratio")
)

var (
	// Uniswap takes total gas of 125k = 21k base gas & 104k swap (this is when user has token balance)
	// Fluid takes total gas of 175k = 21k base gas & 154k swap (this is when user has token balance),
	// with ETH swaps costing less (because no WETH conversion)
	defaultGas = Gas{Swap: 260000}
)
