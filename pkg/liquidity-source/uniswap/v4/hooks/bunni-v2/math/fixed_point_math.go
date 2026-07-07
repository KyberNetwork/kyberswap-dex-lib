package math

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

func Rpow(x *uint256.Int, y int, b *uint256.Int) (*uint256.Int, error) {
	// assembly {
	// 	z := mul(b, iszero(y)) // `0 ** 0 = 1`. Otherwise, `0 ** n = 0`.
	// 	if x {
	// 		z := xor(b, mul(xor(b, x), and(y, 1))) // `z = isEven(y) ? scale : x`
	// 		let half := shr(1, b) // Divide `b` by 2.
	// 		// Divide `y` by 2 every iteration.
	// 		for { y := shr(1, y) } y { y := shr(1, y) } {
	// 			let xx := mul(x, x) // Store x squared.
	// 			let xxRound := add(xx, half) // Round to the nearest number.
	// 			// Revert if `xx + half` overflowed, or if `x ** 2` overflows.
	// 			if or(lt(xxRound, xx), shr(128, x)) {
	// 				mstore(0x00, 0x49f7642b) // `RPowOverflow()`.
	// 				revert(0x1c, 0x04)
	// 			}
	// 			x := div(xxRound, b) // Set `x` to scaled `xxRound`.
	// 			// If `y` is odd:
	// 			if and(y, 1) {
	// 				let zx := mul(z, x) // Compute `z * x`.
	// 				let zxRound := add(zx, half) // Round to the nearest number.
	// 				// If `z * x` overflowed or `zx + half` overflowed:
	// 				if or(xor(div(zx, x), z), lt(zxRound, zx)) {
	// 					// Revert if `x` is non-zero.
	// 					if iszero(iszero(x)) {
	// 						mstore(0x00, 0x49f7642b) // `RPowOverflow()`.
	// 						revert(0x1c, 0x04)
	// 					}
	// 				}
	// 				z := div(zxRound, b) // Return properly scaled `zxRound`.
	// 			}
	// 		}
	// 	}
	// }

	var z uint256.Int
	// z := mul(b, iszero(y)) // `0 ** 0 = 1`. Otherwise, `0 ** n = 0`.
	if y == 0 {
		z.Set(b)
	}

	// if x {
	if !x.IsZero() {
		// z := xor(b, mul(xor(b, x), and(y, 1))) // `z = isEven(y) ? scale : x`
		if (y & 1) == 0 {
			z.Set(b) // isEven(y) ? b : x
		} else {
			z.Set(x)
		}

		// let half := shr(1, b) // Divide `b` by 2.
		var half uint256.Int
		half.Rsh(b, 1)

		// evolving base
		var xv uint256.Int
		xv.Set(x)

		// Divide `y` by 2 every iteration.
		y >>= 1
		for y > 0 {
			// let xx := mul(x, x) // Store x squared.
			var xx uint256.Int
			if _, ov := xx.MulOverflow(&xv, &xv); ov {
				return nil, ErrOverflow
			}

			// Check if upper 128 bits of x are non-zero (shr(128, x))
			var temp uint256.Int
			temp.Rsh(&xv, 128)
			if !temp.IsZero() {
				return nil, ErrOverflow
			}

			// let xxRound := add(xx, half) // Round to the nearest number.
			var xxRound uint256.Int
			if _, ov := xxRound.AddOverflow(&xx, &half); ov {
				return nil, ErrOverflow
			}

			// x := div(xxRound, b) // Set `x` to scaled `xxRound`.
			xv.Div(&xxRound, b)

			// If `y` is odd:
			if (y & 1) == 1 {
				// let zx := mul(z, x) // Compute `z * x`.
				var zx uint256.Int
				if _, ov := zx.MulOverflow(&z, &xv); ov {
					// Revert if `x` is non-zero.
					if !xv.IsZero() {
						return nil, ErrOverflow
					}
				}

				// let zxRound := add(zx, half) // Round to the nearest number.
				var zxRound uint256.Int
				if _, ov := zxRound.AddOverflow(&zx, &half); ov {
					// Revert if `x` is non-zero.
					if !xv.IsZero() {
						return nil, ErrOverflow
					}
				}

				// Check division consistency (xor(div(zx, x), z))
				var divCheck uint256.Int
				if !xv.IsZero() {
					divCheck.Div(&zx, &xv)
					if !divCheck.Eq(&z) {
						return nil, ErrOverflow
					}
				}

				// z := div(zxRound, b) // Return properly scaled `zxRound`.
				z.Div(&zxRound, b)
			}

			y >>= 1
		}
	}

	return &z, nil
}

func MulWad(x, y *uint256.Int) (*uint256.Int, error) {
	// assembly {
	//     // Equivalent to `require(y == 0 || x <= type(uint256).max / y)`.
	//     if mul(y, gt(x, div(not(0), y))) {
	//         mstore(0x00, 0xbac65e5b) // `MulWadFailed()`.
	//         revert(0x1c, 0x04)
	//     }
	//     z := div(mul(x, y), WAD)
	// }

	// Check for overflow: y == 0 || x <= type(uint256).max / y
	if !y.IsZero() {
		var maxDiv uint256.Int
		maxDiv.SetAllOne()
		maxDiv.Div(&maxDiv, y)
		if x.Gt(&maxDiv) {
			return nil, errors.New("MulWadFailed")
		}
	}

	// z := div(mul(x, y), WAD)
	var result uint256.Int
	result.Mul(x, y)
	result.Div(&result, WAD)
	return &result, nil
}

func MulWadUp(x, y *uint256.Int) (*uint256.Int, error) {
	// assembly {
	// 	// Equivalent to `require(y == 0 || x <= type(uint256).max / y)`.
	// 	if mul(y, gt(x, div(not(0), y))) {
	// 		mstore(0x00, 0xbac65e5b) // `MulWadFailed()`.
	// 		revert(0x1c, 0x04)
	// 	}
	// 	z := add(iszero(iszero(mod(mul(x, y), WAD))), div(mul(x, y), WAD))
	// }

	var prod uint256.Int
	_, overflow := prod.MulOverflow(x, y)
	if overflow {
		return nil, ErrOverflow
	}

	var z uint256.Int
	z.Div(&prod, WAD)
	var rem uint256.Int
	rem.Mod(&prod, WAD)
	if !rem.IsZero() {
		z.AddUint64(&z, 1)
		if z.IsZero() {
			return nil, ErrOverflow
		}
	}
	return &z, nil
}

func FullMulDiv(a, b, d *uint256.Int) (*uint256.Int, error) {
	// assembly {
	// 	for {} 1 {} {
	// 		// 512-bit multiply `[p1 p0] = x * y`.
	// 		// Compute the product mod `2**256` and mod `2**256 - 1`
	// 		// then use the Chinese Remainder Theorem to reconstruct
	// 		// the 512 bit result. The result is stored in two 256
	// 		// variables such that `product = p1 * 2**256 + p0`.

	// 		// Least significant 256 bits of the product.
	// 		result := mul(x, y) // Temporarily use `result` as `p0` to save gas.
	// 		let mm := mulmod(x, y, not(0))
	// 		// Most significant 256 bits of the product.
	// 		let p1 := sub(mm, add(result, lt(mm, result)))

	// 		// Handle non-overflow cases, 256 by 256 division.
	// 		if iszero(p1) {
	// 			if iszero(d) {
	// 				mstore(0x00, 0xae47f702) // `FullMulDivFailed()`.
	// 				revert(0x1c, 0x04)
	// 			}
	// 			result := div(result, d)
	// 			break
	// 		}

	// 		// Make sure the result is less than `2**256`. Also prevents `d == 0`.
	// 		if iszero(gt(d, p1)) {
	// 			mstore(0x00, 0xae47f702) // `FullMulDivFailed()`.
	// 			revert(0x1c, 0x04)
	// 		}

	// 		/*------------------- 512 by 256 division --------------------*/

	// 		// Make division exact by subtracting the remainder from `[p1 p0]`.
	// 		// Compute remainder using mulmod.
	// 		let r := mulmod(x, y, d)
	// 		// `t` is the least significant bit of `d`.
	// 		// Always greater or equal to 1.
	// 		let t := and(d, sub(0, d))
	// 		// Divide `d` by `t`, which is a power of two.
	// 		d := div(d, t)
	// 		// Invert `d mod 2**256`
	// 		// Now that `d` is an odd number, it has an inverse
	// 		// modulo `2**256` such that `d * inv = 1 mod 2**256`.
	// 		// Compute the inverse by starting with a seed that is correct
	// 		// correct for four bits. That is, `d * inv = 1 mod 2**4`.
	// 		let inv := xor(2, mul(3, d))
	// 		// Now use Newton-Raphson iteration to improve the precision.
	// 		// Thanks to Hensel's lifting lemma, this also works in modular
	// 		// arithmetic, doubling the correct bits in each step.
	// 		inv := mul(inv, sub(2, mul(d, inv))) // inverse mod 2**8
	// 		inv := mul(inv, sub(2, mul(d, inv))) // inverse mod 2**16
	// 		inv := mul(inv, sub(2, mul(d, inv))) // inverse mod 2**32
	// 		inv := mul(inv, sub(2, mul(d, inv))) // inverse mod 2**64
	// 		inv := mul(inv, sub(2, mul(d, inv))) // inverse mod 2**128
	// 		result :=
	// 			mul(
	// 				// Divide [p1 p0] by the factors of two.
	// 				// Shift in bits from `p1` into `p0`. For this we need
	// 				// to flip `t` such that it is `2**256 / t`.
	// 				or(
	// 					mul(sub(p1, gt(r, result)), add(div(sub(0, t), t), 1)),
	// 					div(sub(result, r), t)
	// 				),
	// 				// inverse mod 2**256
	// 				mul(inv, sub(2, mul(d, inv)))
	// 			)
	// 		break
	// 	}
	// }

	var p0 uint256.Int
	p0.Mul(a, b)

	var mm uint256.Int
	mm.MulMod(a, b, u256.UMax)

	var carry uint64
	if mm.Lt(&p0) {
		carry = 1
	}

	var temp uint256.Int
	temp.AddUint64(&p0, carry)

	var p1 uint256.Int
	p1.Sub(&mm, &temp)

	if p1.IsZero() {
		if d.IsZero() {
			return nil, ErrOverflow
		}
		p0.Div(&p0, d)
		return &p0, nil
	}

	if d.Cmp(&p1) <= 0 {
		return nil, ErrOverflow
	}

	var r uint256.Int
	r.MulMod(a, b, d)

	temp.Sub(u256.U0, d)
	var t uint256.Int
	t.And(d, &temp)

	temp.Div(d, &t)

	var inv uint256.Int
	mm.Mul(u256.U3, &temp)
	inv.Xor(u256.U2, &mm)

	for range 5 {
		mm.Mul(&temp, &inv)
		mm.Sub(u256.U2, &mm)
		inv.Mul(&inv, &mm)
	}

	mm.Mul(&temp, &inv)
	mm.Sub(u256.U2, &mm)
	inv.Mul(&inv, &mm)

	if r.Gt(&p0) {
		p1.SubUint64(&p1, 1)
	}

	mm.Sub(u256.U0, &t).Div(&mm, &t).AddUint64(&mm, 1)

	temp.Sub(&p0, &r).Div(&temp, &t)

	p1.Mul(&p1, &mm).Or(&p1, &temp).Mul(&p1, &inv)

	return &p1, nil
}

func FullMulDivUp(a, b, d *uint256.Int) (*uint256.Int, error) {
	// result = fullMulDiv(x, y, d);
	// /// @solidity memory-safe-assembly
	// assembly {
	// 	if mulmod(x, y, d) {
	// 		result := add(result, 1)
	// 		if iszero(result) {
	// 			mstore(0x00, 0xae47f702) // `FullMulDivFailed()`.
	// 			revert(0x1c, 0x04)
	// 		}
	// 	}
	// }

	z, err := FullMulDiv(a, b, d)
	if err != nil {
		return nil, err
	}
	var rem uint256.Int
	rem.MulMod(a, b, d)
	if !rem.IsZero() {
		z.AddUint64(z, 1)
		if z.IsZero() {
			return nil, ErrFullMulDivFailed
		}
	}
	return z, nil
}
