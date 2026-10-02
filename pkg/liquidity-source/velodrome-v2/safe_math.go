package velodromev2

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

	ErrArithmeticOverflow = errors.New("arithmetic underflow or overflow")
	ErrDivisionByZero     = errors.New("division or modulo by zero")
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

func checkedAdd(z, x, y *uint256.Int) error {
	if _, overflow := z.AddOverflow(x, y); overflow {
		return ErrArithmeticOverflow
	}
	return nil
}

func checkedSub(z, x, y *uint256.Int) error {
	if _, overflow := z.SubOverflow(x, y); overflow {
		return ErrArithmeticOverflow
	}
	return nil
}

func checkedMul(z, x, y *uint256.Int) error {
	if _, overflow := z.MulOverflow(x, y); overflow {
		return ErrArithmeticOverflow
	}
	return nil
}

func checkedDiv(z, x, y *uint256.Int) error {
	if y.IsZero() {
		return ErrDivisionByZero
	}
	z.Div(x, y)
	return nil
}

func checkedMulDiv(z, x, y, d *uint256.Int) error {
	if err := checkedMul(z, x, y); err != nil {
		return err
	}
	return checkedDiv(z, z, d)
}
