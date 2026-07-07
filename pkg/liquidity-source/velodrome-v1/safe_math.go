package velodromev1

import (
	"errors"

	"github.com/holiman/uint256"
)

// a library for performing overflow-safe math, courtesy of DappHub (https://github.com/dapphub/ds-math)

// library SafeMathUniswap {
//     function add(uint x, uint y) internal pure returns (uint z) {
//         require((z = x + y) >= x, 'ds-math-add-overflow');
//     }

//     function sub(uint x, uint y) internal pure returns (uint z) {
//         require((z = x - y) <= x, 'ds-math-sub-underflow');
//     }

//     function mul(uint x, uint y) internal pure returns (uint z) {
//         require(y == 0 || (z = x * y) / y == x, 'ds-math-mul-overflow');
//     }
// }

var (
	ErrDSMathAddOverflow  = errors.New("ds-math-add-overflow")
	ErrDSMathSubUnderflow = errors.New("ds-math-sub-underflow")
	ErrDSMathMulOverflow  = errors.New("ds-math-mul-overflow")
)

func SafeAdd(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Add(x, y)
	if z.Cmp(x) >= 0 {
		return z
	}

	panic(ErrDSMathAddOverflow)
}

func SafeSub(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Sub(x, y)
	if z.Cmp(x) <= 0 {
		return z
	}

	panic(ErrDSMathSubUnderflow)
}

func SafeMul(x, y *uint256.Int) *uint256.Int {
	z := new(uint256.Int).Mul(x, y)
	if y.CmpUint64(0) == 0 || new(uint256.Int).Div(z, y).Cmp(x) == 0 {
		return z
	}

	panic(ErrDSMathMulOverflow)
}
