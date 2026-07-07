package abdkmath64x64

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	// lnConst = 0xB17217F7D1CF79ABC9E3B39803F2F6AF, i.e. ln(2) in 128.128, used to convert a
	// base-2 logarithm into a natural logarithm: ln(x) = log_2(x) * ln(2).
	lnConst = hexU("0xB17217F7D1CF79ABC9E3B39803F2F6AF")

	// msb-detection thresholds for log_2 (unsigned, since the argument is > 0).
	c2p64 = hexU("0x10000000000000000")
	c2p32 = hexU("0x100000000")
	c2p16 = hexU("0x10000")
	c2p8  = hexU("0x100")
	c2p4  = hexU("0x10")
	c2p2  = hexU("0x4")
	c2p1  = hexU("0x2")
)

// Log2 computes the binary logarithm of x in 64.64 fixed point. Reverts for x <= 0.
// Mirrors ABDKMath64x64.log_2 bit-for-bit.
func Log2(x *int256.Int) (int256.Int, error) {
	if x.Sign() <= 0 {
		return int256.Int{}, ErrNonPositive
	}

	xu := asU(x) // x > 0, so this equals x

	// Find the most-significant set bit.
	msb := 0
	var xc uint256.Int
	xc.Set(&xu)
	if xc.Cmp(c2p64) >= 0 {
		xc.Rsh(&xc, 64)
		msb += 64
	}
	if xc.Cmp(c2p32) >= 0 {
		xc.Rsh(&xc, 32)
		msb += 32
	}
	if xc.Cmp(c2p16) >= 0 {
		xc.Rsh(&xc, 16)
		msb += 16
	}
	if xc.Cmp(c2p8) >= 0 {
		xc.Rsh(&xc, 8)
		msb += 8
	}
	if xc.Cmp(c2p4) >= 0 {
		xc.Rsh(&xc, 4)
		msb += 4
	}
	if xc.Cmp(c2p2) >= 0 {
		xc.Rsh(&xc, 2)
		msb += 2
	}
	if xc.Cmp(c2p1) >= 0 {
		msb++
	}

	// result (two's-complement accumulator) = (msb - 64) << 64.
	var resI int256.Int
	resI.SetInt64(int64(msb) - 64)
	resIu := asU(&resI)
	var result uint256.Int
	result.Lsh(&resIu, 64)

	// ux = x << (127 - msb).
	var ux uint256.Int
	ux.Lsh(&xu, uint(127-msb))

	// 64 iterations of the fractional log_2 refinement.
	for shift := 63; shift >= 0; shift-- {
		var sq uint256.Int
		sq.Mul(&ux, &ux) // modular squaring, exactly matching Solidity's `ux *= ux`
		b := sq[3] >> 63 // bit 255 of the product (0 or 1)
		ux.Rsh(&sq, uint(127+b))
		if b == 1 {
			var bitVal uint256.Int
			bitVal.SetUint64(uint64(1) << uint(shift))
			result.Add(&result, &bitVal)
		}
	}

	res := asI(&result)
	return res, nil
}

// Ln computes the natural logarithm of x in 64.64 fixed point. Reverts for x <= 0.
// Mirrors ABDKMath64x64.ln: log_2(x) * ln(2).
func Ln(x *int256.Int) (int256.Int, error) {
	if x.Sign() <= 0 {
		return int256.Int{}, ErrNonPositive
	}

	l2, err := Log2(x)
	if err != nil {
		return int256.Int{}, err
	}

	u := asU(&l2) // two's-complement reinterpretation (log_2 may be negative)
	var prod, sh uint256.Int
	prod.Mul(&u, lnConst) // modular multiply
	sh.Rsh(&prod, 128)
	// The multiply above intentionally overflows for negative log_2; the int128 cast
	// (low 128 bits, sign-extended) recovers the signed natural log.
	res := toInt128(&sh)
	return res, nil
}
