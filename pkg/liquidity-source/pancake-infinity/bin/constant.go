package bin

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "pancake-infinity-bin"

	graphFirstLimit = 1000

	binPoolManagerMethodGetSlot0 = "getSlot0"
	binPoolManagerMethodGetBin   = "getBin"

	_OFFSET_BIN_STEP     = 16
	_SCALE_OFFSET        = 128
	_PROTOCOL_FEE_OFFSET = 24
	_LP_FEE_OFFSET       = 48
	_REAL_ID_SHIFT       = 1 << 23
)

var (
	_ONE                      = uint256.NewInt(1)
	_MAX_LIQUIDITY_PER_BIN, _ = uint256.FromDecimal("65251743116719673010965625540244653191619923014385985379600384103134737")
	_PIPS_DENOMINATOR         = uint256.NewInt(1_000_000)
	_ONE_E12                  = uint256.NewInt(1e12)
	_PRECISION                = uint256.NewInt(1e18) // 1e18
	_SCALE                    = new(uint256.Int).Lsh(_ONE, _SCALE_OFFSET)
	_BASIS_POINT_MAX          = uint256.NewInt(10_000)
	_POW_U                    = uint256.NewInt(0x100000)
	_MASK12                   = uint256.NewInt(0xfff)
	_MASK16                   = uint256.NewInt(0xffff)
	_MASK24                   = uint256.NewInt(0xffffff)

	ErrLiquidityOverflow             = errors.New("BinHelper__LiquidityOverflow")
	ErrMaxLiquidityPerBin            = errors.New("BinPool__MaxLiquidityPerBinExceeded")
	ErrInsufficientAmountUnSpecified = errors.New("BinPool__InsufficientAmountUnSpecified")
	ErrBinIDNotFound                 = errors.New("binId not found")
	ErrPowUnderflow                  = errors.New("pow underflow")
	ErrMulDivOverflow                = errors.New("mul div overflow")
	ErrMulShiftOverflow              = errors.New("mul shift overflow")
)
