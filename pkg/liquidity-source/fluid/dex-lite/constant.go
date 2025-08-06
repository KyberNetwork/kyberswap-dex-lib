package dexLite

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
	// Storage slots for FluidDexLite contract (matches Variables.sol layout)
	StorageSlotIsAuth           = 0 // Slot 0: _isAuth mapping
	StorageSlotDexesList        = 1 // Slot 1: _dexesList array
	StorageSlotDexVariables     = 2 // Slot 2: _dexVariables mapping
	StorageSlotCenterPriceShift = 3 // Slot 3: _centerPriceShift mapping
	StorageSlotRangeShift       = 4 // Slot 4: _rangeShift mapping
	StorageSlotThresholdShift   = 5 // Slot 5: _thresholdShift mapping

	// StorageSlotDexList - Legacy storage slot (kept for compatibility)
	StorageSlotDexList = "0x1" // Slot 1: dex list array (_dexesList)

	FeePercentPrecision     float64 = 1e4 // 10000
	TokensDecimalsPrecision         = 9   // FluidDexLite uses 9 decimal precision internally
	DefaultExponentSize             = 8

	defaultGas = 82651
)

// Bit positions for DexVariables (from contract DexLiteSlotsLink.sol)
const (
	BitPosFee                         = 0
	BitPosRevenueCut                  = 13
	BitPosRebalancingStatus           = 20
	BitPosCenterPriceShiftActive      = 22
	BitPosCenterPrice                 = 23
	BitPosCenterPriceContractAddress  = 63
	BitPosRangePercentShiftActive     = 82
	BitPosUpperPercent                = 83
	BitPosLowerPercent                = 97
	BitPosThresholdPercentShiftActive = 111
	BitPosUpperShiftThresholdPercent  = 112
	BitPosLowerShiftThresholdPercent  = 119
	BitPosToken0Decimals              = 126
	BitPosToken1Decimals              = 131
	BitPosToken0TotalSupplyAdjusted   = 136
	BitPosToken1TotalSupplyAdjusted   = 196
)

// Bit positions for CenterPriceShift
const (
	BitPosCenterPriceShiftLastInteractionTimestamp = 0
	BitPosCenterPriceShiftShiftingTime             = 33
	BitPosCenterPriceShiftMaxCenterPrice           = 57
	BitPosCenterPriceShiftMinCenterPrice           = 85
	BitPosCenterPriceShiftPercent                  = 113
	BitPosCenterPriceShiftTimeToShift              = 133
	BitPosCenterPriceShiftTimestamp                = 153
)

// Bit positions for RangeShift
const (
	BitPosRangeShiftOldUpperRangePercent = 0
	BitPosRangeShiftOldLowerRangePercent = 14
	BitPosRangeShiftTimeToShift          = 28
	BitPosRangeShiftTimestamp            = 48
)

// Bit positions for ThresholdShift
const (
	BitPosThresholdShiftOldUpperThresholdPercent = 0
	BitPosThresholdShiftOldLowerThresholdPercent = 7
	BitPosThresholdShiftTimeToShift              = 14
	BitPosThresholdShiftTimestamp                = 34
)

// Bit masks for extracting values
var (
	X1   = uint256.NewInt(0x1)
	X2   = uint256.NewInt(0x3)
	X7   = uint256.NewInt(0x7f)
	X13  = uint256.NewInt(0x1fff)
	X13B = big.NewInt(0x1fff)
	X14  = uint256.NewInt(0x3fff)
	X19  = uint256.NewInt(0x7ffff)
	X20  = uint256.NewInt(0xfffff)
	X24  = uint256.NewInt(0xffffff)
	X28  = uint256.NewInt(0xfffffff)
	X33  = uint256.NewInt(0x1ffffffff)
	X40  = uint256.NewInt(0xffffffffff)
	X60  = uint256.NewInt(0xfffffffffffffff)
)

// Constants
var (
	TwoDecimals          = uint256.NewInt(100)
	FourDecimals         = uint256.NewInt(10000)
	SixDecimals          = uint256.NewInt(1000000)
	PricePrecision       = big256.TenPow(27) // 1e27
	PricePrecisionSq     = big256.TenPow(54) // 1e54
	MinimumLiquiditySwap = FourDecimals
	DefaultExponentMask  = uint256.NewInt(0xff)
	threshold1e38        = big256.TenPow(38)
)

// Error definitions
var (
	ErrInvalidAmountIn  = errors.New("invalid amountIn")
	ErrInvalidAmountOut = errors.New("invalid amount out")
	ErrInvalidToken     = errors.New("invalid token")

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
