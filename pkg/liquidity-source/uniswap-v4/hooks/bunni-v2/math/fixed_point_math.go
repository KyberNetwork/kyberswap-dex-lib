package math

import (
	"log"

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
				log.Fatalln(0, x, y, b)

				return nil, ErrOverflow
			}

			// Check if upper 128 bits of x are non-zero (shr(128, x))
			var temp uint256.Int
			temp.Rsh(&xv, 128)
			if !temp.IsZero() {
				log.Fatalln(1, x, y, b)

				return nil, ErrOverflow
			}

			// let xxRound := add(xx, half) // Round to the nearest number.
			var xxRound uint256.Int
			if _, ov := xxRound.AddOverflow(&xx, &half); ov {
				log.Fatalln(2, x, y, b)

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
						log.Fatalln(3, x, y, b)

						return nil, ErrOverflow
					}
				}

				// let zxRound := add(zx, half) // Round to the nearest number.
				var zxRound uint256.Int
				if _, ov := zxRound.AddOverflow(&zx, &half); ov {
					// Revert if `x` is non-zero.
					if !xv.IsZero() {
						log.Fatalln(4, x, y, b)

						return nil, ErrOverflow
					}
				}

				// Check division consistency (xor(div(zx, x), z))
				var divCheck uint256.Int
				if !xv.IsZero() {
					divCheck.Div(&zx, &xv)
					if !divCheck.Eq(&z) {
						log.Fatalln(5, x, y, b)

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

	// Least significant 256 bits of the product.
	var p0 uint256.Int
	p0.Mul(a, b) // result := mul(x, y)

	// let mm := mulmod(x, y, not(0))
	var mm uint256.Int
	mm.MulMod(a, b, u256.UMax)

	// Most significant 256 bits of the product.
	// let p1 := sub(mm, add(result, lt(mm, result)))
	var p1 uint256.Int
	var carry uint64
	if mm.Lt(&p0) {
		carry = 1
	}
	var temp uint256.Int
	temp.Set(&p0)
	temp.AddUint64(&temp, carry)
	p1.Sub(&mm, &temp)

	// Handle non-overflow cases, 256 by 256 division.
	if p1.IsZero() {
		if d.IsZero() {
			return nil, ErrOverflow // FullMulDivFailed()
		}
		var result uint256.Int
		result.Div(&p0, d)
		return &result, nil
	}

	// Make sure the result is less than `2**256`. Also prevents `d == 0`.
	// if iszero(gt(d, p1)) means if !(d > p1), i.e., if d <= p1
	if d.Cmp(&p1) <= 0 {
		return nil, ErrOverflow // FullMulDivFailed()
	}

	/*------------------- 512 by 256 division --------------------*/

	// Make division exact by subtracting the remainder from `[p1 p0]`.
	// Compute remainder using mulmod.
	var r uint256.Int
	r.MulMod(a, b, d)

	// `t` is the least significant bit of `d`.
	// t := and(d, sub(0, d))
	var negD uint256.Int
	negD.Sub(u256.U0, d) // sub(0, d)
	var t uint256.Int
	t.And(d, &negD)

	// Divide `d` by `t`, which is a power of two.
	var dOdd uint256.Int
	dOdd.Div(d, &t)

	// Invert `d mod 2**256`
	// let inv := xor(2, mul(3, d))
	var inv uint256.Int
	var temp3d uint256.Int
	temp3d.Mul(u256.U3, &dOdd)
	inv.Xor(u256.U2, &temp3d)

	// Newton-Raphson iterations (5 iterations in loop + 1 final)
	for range 5 {
		var tmp uint256.Int
		tmp.Mul(&dOdd, &inv)
		tmp.Sub(u256.U2, &tmp)
		inv.Mul(&inv, &tmp)
	}
	// Final iteration: mul(inv, sub(2, mul(d, inv)))
	var finalTmp uint256.Int
	finalTmp.Mul(&dOdd, &inv)
	finalTmp.Sub(u256.U2, &finalTmp)
	inv.Mul(&inv, &finalTmp)

	// Assembly calculation:
	// result := mul(
	//     or(
	//         mul(sub(p1, gt(r, result)), add(div(sub(0, t), t), 1)),
	//         div(sub(result, r), t)
	//     ),
	//     inv
	// )

	// First part: sub(p1, gt(r, result))
	var p1Adj uint256.Int
	p1Adj.Set(&p1)
	if r.Gt(&p0) {
		p1Adj.SubUint64(&p1Adj, 1)
	}

	// Second part: add(div(sub(0, t), t), 1)
	var negT uint256.Int
	negT.Sub(u256.U0, &t)
	var divNegT uint256.Int
	divNegT.Div(&negT, &t)
	divNegT.AddUint64(&divNegT, 1)

	// Third part: div(sub(result, r), t)
	var p0MinusR uint256.Int
	p0MinusR.Sub(&p0, &r)
	var divP0R uint256.Int
	divP0R.Div(&p0MinusR, &t)

	// Combine: mul(p1Adj, divNegT)
	var leftPart uint256.Int
	leftPart.Mul(&p1Adj, &divNegT)

	// OR operation: or(leftPart, divP0R)
	var combined uint256.Int
	combined.Or(&leftPart, &divP0R)

	// Final multiplication with inverse
	var result uint256.Int
	result.Mul(&combined, &inv)

	return &result, nil
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
