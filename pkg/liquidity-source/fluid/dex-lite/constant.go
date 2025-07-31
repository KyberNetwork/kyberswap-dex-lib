package dexLite

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "fluid-dex-lite"
)

const (
	// FluidDexLite contract methods
	DexLiteMethodSwapSingle = "swapSingle"

	// ERC20 Token methods
	TokenMethodDecimals = "decimals"

	// StorageRead methods
	SRMethodReadFromStorage = "readFromStorage"
)

const (
	// Storage slots for FluidDexLite
	StorageSlotDexList = "0x1" // Slot 1: dex list array (_dexesList)

	// Fee precision
	FeePercentPrecision float64 = 1e4 // 10000

	// Minimum liquidity for swaps
	MinimumLiquiditySwap = 10000
)

// Bit positions for DexVariables (from contract DexLiteSlotsLink.sol)
const (
	BitsDexLiteDexVariablesFee                         = 0
	BitsDexLiteDexVariablesRevenueCut                  = 13
	BitsDexLiteDexVariablesRebalancingStatus           = 20
	BitsDexLiteDexVariablesCenterPriceShiftActive      = 22
	BitsDexLiteDexVariablesCenterPrice                 = 23
	BitsDexLiteDexVariablesCenterPriceContractAddress  = 63
	BitsDexLiteDexVariablesRangePercentShiftActive     = 82
	BitsDexLiteDexVariablesUpperPercent                = 83
	BitsDexLiteDexVariablesLowerPercent                = 97
	BitsDexLiteDexVariablesThresholdPercentShiftActive = 111
	BitsDexLiteDexVariablesUpperShiftThresholdPercent  = 112
	BitsDexLiteDexVariablesLowerShiftThresholdPercent  = 119
	BitsDexLiteDexVariablesToken0Decimals              = 126
	BitsDexLiteDexVariablesToken1Decimals              = 131
	BitsDexLiteDexVariablesToken0TotalSupplyAdjusted   = 136
	BitsDexLiteDexVariablesToken1TotalSupplyAdjusted   = 196
)

// Bit positions for CenterPriceShift
const (
	BitsDexLiteCenterPriceShiftLastInteractionTimestamp = 0
	BitsDexLiteCenterPriceShiftShiftingTime             = 33
	BitsDexLiteCenterPriceShiftMaxCenterPrice           = 57
	BitsDexLiteCenterPriceShiftMinCenterPrice           = 85
	BitsDexLiteCenterPriceShiftPercent                  = 113
	BitsDexLiteCenterPriceShiftTimeToShift              = 133
	BitsDexLiteCenterPriceShiftTimestamp                = 153
)

// Bit positions for RangeShift
const (
	BitsDexLiteRangeShiftOldUpperRangePercent = 0
	BitsDexLiteRangeShiftOldLowerRangePercent = 14
	BitsDexLiteRangeShiftTimeToShift          = 28
	BitsDexLiteRangeShiftTimestamp            = 48
)

// Bit positions for ThresholdShift
const (
	BitsDexLiteThresholdShiftOldUpperThresholdPercent = 0
	BitsDexLiteThresholdShiftOldLowerThresholdPercent = 7
	BitsDexLiteThresholdShiftTimeToShift              = 14
	BitsDexLiteThresholdShiftTimestamp                = 34
)

// Bit masks for extracting values
var (
	X1  = big.NewInt(0x1)
	X2  = big.NewInt(0x3)
	X5  = big.NewInt(0x1f)
	X7  = big.NewInt(0x7f)
	X13 = big.NewInt(0x1fff)
	X14 = big.NewInt(0x3fff)
	X19 = big.NewInt(0x7ffff)
	X20 = big.NewInt(0xfffff)
	X24 = big.NewInt(0xffffff)
	X28 = big.NewInt(0xfffffff)
	X33 = big.NewInt(0x1ffffffff)
	X40 = big.NewInt(0xffffffffff)
	X60 = big.NewInt(0xfffffffffffffff)
	X64 = new(big.Int).SetBytes([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	X73 = new(big.Int).SetBytes([]byte{0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
)

// Constants
var (
	TwoDecimals             = big.NewInt(100)
	FourDecimals            = big.NewInt(10000)
	SixDecimals             = big.NewInt(1000000)
	PricePrecision          = bignumber.TenPowInt(27) // 1e27
	TokensDecimalsPrecision = uint64(9)               // FluidDexLite uses 9 decimal precision internally
	DefaultExponentSize     = uint64(8)
	DefaultExponentMask     = big.NewInt(0xff)
)

// Error definitions
var (
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")

	ErrInsufficientReserve    = errors.New("insufficient reserve: tokenOut amount exceeds reserve")
	ErrSwapAndArbitragePaused = errors.New("swap and arbitrage paused")

	ErrInsufficientLiquidity = errors.New("insufficient liquidity for swap")
	ErrInvalidPoolState      = errors.New("invalid pool state")
	ErrPoolNotFound          = errors.New("pool not found")

	ErrInvalidFee                      = errors.New("invalid fee")
	ErrInvalidCenterPrice              = errors.New("invalid center price")
	ErrPoolNotInitialized              = errors.New("pool not initialized")
	ErrExcessiveSwapAmount             = errors.New("excessive swap amount")
	ErrTokenReservesRatioTooHigh       = errors.New("token reserves ratio too high")
	ErrAdjustedSupplyOverflow          = errors.New("adjusted supply overflow")
	ErrInvalidFeeRate                  = errors.New("invalid fee rate")
	ErrExternalCenterPriceNotSupported = errors.New("external center price feeds not supported")
)

// Gas costs
var (
	// FluidDexLite is highly optimized and only takes 10K gas
	defaultGas = Gas{Swap: 10000}
)
