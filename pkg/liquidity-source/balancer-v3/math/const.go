package math

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	ErrAddOverflow  = errors.New("ADD_OVERFLOW")
	ErrSubOverflow  = errors.New("SUB_OVERFLOW")
	ErrMulOverflow  = errors.New("MUL_OVERFLOW")
	ErrZeroDivision = errors.New("ZERO_DIVISION")

	ErrBaseOutOfBounds                    = errors.New("Base_OutOfBounds")
	ErrExponentOutOfBounds                = errors.New("Exponent_OutOfBounds")
	ErrProductOutOfBounds                 = errors.New("Product_OutOfBounds")
	ErrStableInvariantDidNotConverge      = errors.New("stable invariant didn't converge")
	ErrStableComputeBalanceDidNotConverge = errors.New("stable computeBalance didn't converge")

	ErrMaxInRatio  = errors.New("MAX_IN_RATIO")
	ErrMaxOutRatio = errors.New("MAX_OUT_RATIO")

	U0       = uint256.NewInt(0)
	U1       = uint256.NewInt(1)
	U2       = uint256.NewInt(2)
	U5       = uint256.NewInt(5)
	U1e18    = uint256.NewInt(1e18)
	U2e18    = uint256.NewInt(2e18)
	U4e18    = uint256.NewInt(4e18)
	U1e20, _ = uint256.FromDecimal("100000000000000000000")
	U2p254   = new(uint256.Int).Lsh(U1, 254) // 2^254

	UMaxPowRelativeError = uint256.NewInt(10000)               // 10^(-14)
	UMildExponentBound   = new(uint256.Int).Div(U2p254, U1e20) // 2^254 / uint256(ONE_20)
	UAmpPrecision        = uint256.NewInt(1e3)
	UMaxInRatio          = uint256.NewInt(30e16)
	UMaxOutRatio         = uint256.NewInt(30e16)

	i0       = int256.NewInt(0)
	i1       = int256.NewInt(1)
	I2       = int256.NewInt(2)
	i3       = int256.NewInt(3)
	i5       = int256.NewInt(5)
	i7       = int256.NewInt(7)
	i9       = int256.NewInt(9)
	i11      = int256.NewInt(11)
	i13      = int256.NewInt(13)
	i15      = int256.NewInt(15)
	i20      = int256.NewInt(20)
	i40      = int256.NewInt(40)
	i100     = int256.NewInt(100)
	i1e9     = int256.NewInt(1e9)
	i1e17    = int256.NewInt(1e17)
	i1e18    = int256.NewInt(1e18)
	i1e20, _ = int256.FromDec("100000000000000000000")
	i1e36, _ = int256.FromDec("1000000000000000000000000000000000000")
	i1e38, _ = int256.FromDec("100000000000000000000000000000000000000")

	// 18 decimal constants
	iX0, _ = int256.FromDec("128000000000000000000")                                    // 2ˆ7
	iA0, _ = int256.FromDec("38877084059945950922200000000000000000000000000000000000") // eˆ(x0) (no decimals)
	iX1, _ = int256.FromDec("64000000000000000000")                                     // 2^6
	iA1, _ = int256.FromDec("6235149080811616882910000000")                             // eˆ(x1) (no decimals)

	// 20 decimal constants
	iX2, _  = int256.FromDec("3200000000000000000000")             // 2^5
	iA2, _  = int256.FromDec("7896296018268069516100000000000000") // eˆ(x2)
	iX3, _  = int256.FromDec("1600000000000000000000")             // 2ˆ4
	iA3, _  = int256.FromDec("888611052050787263676000000")        // eˆ(x3)
	iX4, _  = int256.FromDec("800000000000000000000")              // 2ˆ3
	iA4, _  = int256.FromDec("298095798704172827474000")           // eˆ(x4)
	iX5, _  = int256.FromDec("400000000000000000000")              // 2ˆ2
	iA5, _  = int256.FromDec("5459815003314423907810")             // eˆ(x5)
	iX6, _  = int256.FromDec("200000000000000000000")              // 2ˆ1
	iA6, _  = int256.FromDec("738905609893065022723")              // eˆ(x6)
	iX7, _  = int256.FromDec("100000000000000000000")              // 2ˆ0
	iA7, _  = int256.FromDec("271828182845904523536")              // eˆ(x7)
	iX8, _  = int256.FromDec("50000000000000000000")               // 2ˆ-1
	iA8, _  = int256.FromDec("164872127070012814685")              // eˆ(x8)
	iX9, _  = int256.FromDec("25000000000000000000")               // 2ˆ-2
	iA9, _  = int256.FromDec("128402541668774148407")              // eˆ(x9)
	iX10, _ = int256.FromDec("12500000000000000000")               // 2ˆ-3
	iA10, _ = int256.FromDec("113314845306682631683")              // eˆ(x10)
	iX11, _ = int256.FromDec("6250000000000000000")                // 2ˆ-4
	iA11, _ = int256.FromDec("106449445891785942956")              // eˆ(x11)

	ILn36LowerBound     = new(int256.Int).Sub(i1e18, i1e17)                        // ONE_18 - 1e17
	ILn36UpperBound     = new(int256.Int).Add(i1e18, i1e17)                        // ONE_18 + 1e17
	IMaxNaturalExponent = new(int256.Int).Mul(int256.NewInt(130), i1e18)           // 130e18
	IMinNaturalExponent = new(int256.Int).Mul(int256.NewInt(-41), i1e18)           // -41e18
	IMaxBalances, _     = int256.FromDec("10000000000000000000000000000000000")    // 1e34
	IMaxInvariant, _    = int256.FromDec("10000000000000000000000000000000000000") // 1e37
)
