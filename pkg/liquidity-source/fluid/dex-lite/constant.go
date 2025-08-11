package dexLite

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "fluid-dex-lite"

	SRMethodReadFromStorage = "readFromStorage"

	FeePercentPrecision     float64 = 1e4 // 10000
	TokensDecimalsPrecision         = 9   // FluidDexLite uses 9 decimal precision internally
	DefaultExponentSize             = 8

	defaultGas = 82651

	MaxBatchSize = 100

	// Bit positions for DexVariables (from contract DexLiteSlotsLink.sol)

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

	// Bit positions for CenterPriceShift

	BitPosCenterPriceShiftLastInteractionTimestamp = 0
	BitPosCenterPriceShiftShiftingTime             = 33
	BitPosCenterPriceShiftMaxCenterPrice           = 57
	BitPosCenterPriceShiftMinCenterPrice           = 85
	BitPosCenterPriceShiftPercent                  = 113
	BitPosCenterPriceShiftTimeToShift              = 133
	BitPosCenterPriceShiftTimestamp                = 153

	// Bit positions for RangeShift

	BitPosRangeShiftOldUpperRangePercent = 0
	BitPosRangeShiftOldLowerRangePercent = 14
	BitPosRangeShiftTimeToShift          = 28
	BitPosRangeShiftTimestamp            = 48

	// Bit positions for ThresholdShift

	BitPosThresholdShiftOldUpperThresholdPercent = 0
	BitPosThresholdShiftOldLowerThresholdPercent = 7
	BitPosThresholdShiftTimeToShift              = 14
	BitPosThresholdShiftTimestamp                = 34
)

var (
	// Storage slots for FluidDexLite contract (matches Variables.sol layout)

	StorageSlotIsAuth           = common.HexToHash("0x0") // _isAuth mapping
	StorageSlotDexesList        = common.HexToHash("0x1") // _dexesList array
	StorageSlotDexVariables     = common.HexToHash("0x2") // _dexVariables mapping
	StorageSlotCenterPriceShift = common.HexToHash("0x3") // _centerPriceShift mapping
	StorageSlotRangeShift       = common.HexToHash("0x4") // _rangeShift mapping
	StorageSlotThresholdShift   = common.HexToHash("0x5") // _thresholdShift mapping

	addressPadding = make([]byte, 12)
	bytes8Padding  = make([]byte, 24)

	// Bit masks for extracting values

	X1  = uint256.NewInt(0x1)
	X2  = uint256.NewInt(0x3)
	X7  = uint256.NewInt(0x7f)
	X13 = uint256.NewInt(0x1fff)
	X14 = uint256.NewInt(0x3fff)
	X19 = uint256.NewInt(0x7ffff)
	X20 = uint256.NewInt(0xfffff)
	X24 = uint256.NewInt(0xffffff)
	X28 = uint256.NewInt(0xfffffff)
	X33 = uint256.NewInt(0x1ffffffff)
	X40 = uint256.NewInt(0xffffffffff)
	X60 = uint256.NewInt(0xfffffffffffffff)

	TwoDecimals          = uint256.NewInt(100)
	FourDecimals         = uint256.NewInt(10000)
	SixDecimals          = uint256.NewInt(1000000)
	PricePrecision       = big256.TenPow(27) // 1e27
	PricePrecisionSq     = big256.TenPow(54) // 1e54
	MinimumLiquiditySwap = FourDecimals
	DefaultExponentMask  = uint256.NewInt(0xff)
	threshold1e38        = big256.TenPow(38)

	// Error definitions

	ErrInvalidAmountIn                 = errors.New("invalid amountIn")
	ErrInvalidAmountOut                = errors.New("invalid amount out")
	ErrInvalidToken                    = errors.New("invalid token")
	ErrInsufficientReserve             = errors.New("insufficient reserve: tokenOut amount exceeds reserve")
	ErrPoolNotInitialized              = errors.New("pool not initialized")
	ErrExcessiveSwapAmount             = errors.New("excessive swap amount")
	ErrTokenReservesRatioTooHigh       = errors.New("token reserves ratio too high")
	ErrAdjustedSupplyOverflow          = errors.New("adjusted supply overflow")
	ErrInvalidFeeRate                  = errors.New("invalid fee rate")
	ErrExternalCenterPriceNotSupported = errors.New("external center price feeds not supported")
)
