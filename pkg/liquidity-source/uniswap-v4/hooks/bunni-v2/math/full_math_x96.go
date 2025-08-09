package math

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

var (
	ErrFullMulDivFailed = errors.New("FullMulDivFailed")
	ErrMulDivFailed     = errors.New("MulDivFailed")
)

// FullMulX96 calculates `floor(x * y / 2 ** 96)` with full precision
func FullMulX96(a, b *uint256.Int) (*uint256.Int, error) {
	// assembly {
	// 	// Temporarily use `z` as `p0` to save gas.
	// 	z := mul(x, y) // Lower 256 bits of `x * y`. We'll call this `z`.
	// 	for {} 1 {} {
	// 		if iszero(or(iszero(x), eq(div(z, x), y))) {
	// 			let mm := mulmod(x, y, not(0))
	// 			let p1 := sub(mm, add(z, lt(mm, z))) // Upper 256 bits of `x * y`.
	// 			//         |      p1     |      z     |
	// 			// Before: | p1_0 ¦ p1_1 | z_0  ¦ z_1 |
	// 			// Final:  |   0  ¦ p1_0 | p1_1 ¦ z_0 |
	// 			// Check that final `z` doesn't overflow by checking that p1_0 = 0.
	// 			if iszero(shr(96, p1)) {
	// 				z := add(shl(160, p1), shr(96, z))
	// 				break
	// 			}
	// 			mstore(0x00, 0xae47f702) // `FullMulDivFailed()`.
	// 			revert(0x1c, 0x04)
	// 		}
	// 		z := shr(96, z)
	// 		break
	// 	}
	// }

	var z uint256.Int
	z.Mul(a, b)

	var divCheck uint256.Int
	divCheck.Div(&z, a)
	if a.IsZero() || divCheck.Eq(b) {
		z.Rsh(&z, 96)
		return &z, nil
	}

	var mm, p1 uint256.Int
	mm.MulMod(a, b, u256.UMax)
	p1.Sub(&mm, &z)
	if mm.Lt(&z) {
		p1.SubUint64(&p1, 1)
	}

	var top uint256.Int
	top.Rsh(&p1, 96)
	if top.Sign() != 0 {
		return nil, ErrFullMulDivFailed
	}

	z.Rsh(&z, 96)
	p1.Lsh(&p1, 160)
	z.Or(&z, &p1)
	return &z, nil
}

// FullMulX96Up calculates ceil(a * b / 2^96) with full precision
func FullMulX96Up(a, b *uint256.Int) (*uint256.Int, error) {
	// z = fullMulX96(x, y);
	// /// @solidity memory-safe-assembly
	// assembly {
	// 	if mulmod(x, y, Q96) {
	// 		z := add(z, 1)
	// 		if iszero(z) {
	// 			mstore(0x00, 0xae47f702) // `FullMulDivFailed()`.
	// 			revert(0x1c, 0x04)
	// 		}
	// 	}
	// }

	z, err := FullMulX96(a, b)
	if err != nil {
		return nil, err
	}

	var remainder uint256.Int
	remainder.MulMod(a, b, Q96)
	if !remainder.IsZero() {
		z, _ = new(uint256.Int).AddOverflow(z, u256.U1)
		if z.IsZero() {
			return nil, ErrMulDivFailed
		}
	}
	return z, nil
}
