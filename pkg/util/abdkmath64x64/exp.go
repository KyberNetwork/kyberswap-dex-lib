package abdkmath64x64

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

var (
	// exp2Limit = 0x400000000000000000 (2^70). Both exp and exp_2 revert for x >= exp2Limit
	// and return 0 for x < -exp2Limit.
	exp2Limit    = hexI("0x400000000000000000")
	negExp2Limit = new(int256.Int).Neg(hexI("0x400000000000000000"))

	// twoPow127 = 2^127, the exp_2 seed (0x80000000000000000000000000000000).
	twoPow127 = hexU("0x80000000000000000000000000000000")

	// expConst = 0x171547652B82FE1777D0FFDA0D23A7D12, i.e. log2(e) in 128.128, used by exp to
	// convert a natural exponent into a base-2 exponent: exp(x) = exp_2(x * log2(e)).
	expConst = hexI("0x171547652B82FE1777D0FFDA0D23A7D12")

	// exp2Magic are the exp_2 per-bit multipliers, indexed from the most significant tested
	// fractional bit (2^63) down to 2^0. Copied verbatim from ABDKMath64x64.exp_2; each is a
	// 128.128 fixed-point representation of 2^(1/2^k).
	exp2Magic = [64]*uint256.Int{
		hexU("0x16A09E667F3BCC908B2FB1366EA957D3E"), // bit 63 (2^-1)
		hexU("0x1306FE0A31B7152DE8D5A46305C85EDEC"), // bit 62 (2^-2)
		hexU("0x1172B83C7D517ADCDF7C8C50EB14A791F"), // bit 61
		hexU("0x10B5586CF9890F6298B92B71842A98363"), // bit 60
		hexU("0x1059B0D31585743AE7C548EB68CA417FD"), // bit 59
		hexU("0x102C9A3E778060EE6F7CACA4F7A29BDE8"), // bit 58
		hexU("0x10163DA9FB33356D84A66AE336DCDFA3F"), // bit 57
		hexU("0x100B1AFA5ABCBED6129AB13EC11DC9543"), // bit 56
		hexU("0x10058C86DA1C09EA1FF19D294CF2F679B"), // bit 55
		hexU("0x1002C605E2E8CEC506D21BFC89A23A00F"), // bit 54
		hexU("0x100162F3904051FA128BCA9C55C31E5DF"), // bit 53
		hexU("0x1000B175EFFDC76BA38E31671CA939725"), // bit 52
		hexU("0x100058BA01FB9F96D6CACD4B180917C3D"), // bit 51
		hexU("0x10002C5CC37DA9491D0985C348C68E7B3"), // bit 50
		hexU("0x1000162E525EE054754457D5995292026"), // bit 49
		hexU("0x10000B17255775C040618BF4A4ADE83FC"), // bit 48
		hexU("0x1000058B91B5BC9AE2EED81E9B7D4CFAB"), // bit 47
		hexU("0x100002C5C89D5EC6CA4D7C8ACC017B7C9"), // bit 46
		hexU("0x10000162E43F4F831060E02D839A9D16D"), // bit 45
		hexU("0x100000B1721BCFC99D9F890EA06911763"), // bit 44
		hexU("0x10000058B90CF1E6D97F9CA14DBCC1628"), // bit 43
		hexU("0x1000002C5C863B73F016468F6BAC5CA2B"), // bit 42
		hexU("0x100000162E430E5A18F6119E3C02282A5"), // bit 41
		hexU("0x1000000B1721835514B86E6D96EFD1BFE"), // bit 40
		hexU("0x100000058B90C0B48C6BE5DF846C5B2EF"), // bit 39
		hexU("0x10000002C5C8601CC6B9E94213C72737A"), // bit 38
		hexU("0x1000000162E42FFF037DF38AA2B219F06"), // bit 37
		hexU("0x10000000B17217FBA9C739AA5819F44F9"), // bit 36
		hexU("0x1000000058B90BFCDEE5ACD3C1CEDC823"), // bit 35
		hexU("0x100000002C5C85FE31F35A6A30DA1BE50"), // bit 34
		hexU("0x10000000162E42FF0999CE3541B9FFFCF"), // bit 33
		hexU("0x100000000B17217F80F4EF5AADDA45554"), // bit 32
		hexU("0x10000000058B90BFBF8479BD5A81B51AD"), // bit 31
		hexU("0x1000000002C5C85FDF84BD62AE30A74CC"), // bit 30
		hexU("0x100000000162E42FEFB2FED257559BDAA"), // bit 29
		hexU("0x1000000000B17217F7D5A7716BBA4A9AE"), // bit 28
		hexU("0x100000000058B90BFBE9DDBAC5E109CCE"), // bit 27
		hexU("0x10000000002C5C85FDF4B15DE6F17EB0D"), // bit 26
		hexU("0x1000000000162E42FEFA494F1478FDE05"), // bit 25
		hexU("0x10000000000B17217F7D20CF927C8E94C"), // bit 24
		hexU("0x1000000000058B90BFBE8F71CB4E4B33D"), // bit 23
		hexU("0x100000000002C5C85FDF477B662B26945"), // bit 22
		hexU("0x10000000000162E42FEFA3AE53369388C"), // bit 21
		hexU("0x100000000000B17217F7D1D351A389D40"), // bit 20
		hexU("0x10000000000058B90BFBE8E8B2D3D4EDE"), // bit 19
		hexU("0x1000000000002C5C85FDF4741BEA6E77E"), // bit 18
		hexU("0x100000000000162E42FEFA39FE95583C2"), // bit 17
		hexU("0x1000000000000B17217F7D1CFB72B45E1"), // bit 16
		hexU("0x100000000000058B90BFBE8E7CC35C3F0"), // bit 15
		hexU("0x10000000000002C5C85FDF473E242EA38"), // bit 14
		hexU("0x1000000000000162E42FEFA39F02B772C"), // bit 13
		hexU("0x10000000000000B17217F7D1CF7D83C1A"), // bit 12
		hexU("0x1000000000000058B90BFBE8E7BDCBE2E"), // bit 11
		hexU("0x100000000000002C5C85FDF473DEA871F"), // bit 10
		hexU("0x10000000000000162E42FEFA39EF44D91"), // bit 9
		hexU("0x100000000000000B17217F7D1CF79E949"), // bit 8
		hexU("0x10000000000000058B90BFBE8E7BCE544"), // bit 7
		hexU("0x1000000000000002C5C85FDF473DE6ECA"), // bit 6
		hexU("0x100000000000000162E42FEFA39EF366F"), // bit 5
		hexU("0x1000000000000000B17217F7D1CF79AFA"), // bit 4
		hexU("0x100000000000000058B90BFBE8E7BCD6D"), // bit 3
		hexU("0x10000000000000002C5C85FDF473DE6B2"), // bit 2
		hexU("0x1000000000000000162E42FEFA39EF358"), // bit 1
		hexU("0x10000000000000000B17217F7D1CF79AB"), // bit 0 (2^-64)
	}
)

// Exp2 computes 2^x in 64.64 fixed point. Reverts on overflow (x >= 2^70); returns 0 for
// x < -2^70. Mirrors ABDKMath64x64.exp_2 bit-for-bit.
func Exp2(x *int256.Int) (int256.Int, error) {
	if x.Cmp(exp2Limit) >= 0 {
		return int256.Int{}, ErrOverflow
	}
	if x.Cmp(negExp2Limit) < 0 {
		return int256.Int{}, nil // underflow -> 0
	}

	var result uint256.Int
	result.Set(twoPow127)

	xb := asU(x) // raw two's-complement bits; only the low 64 fractional bits are tested
	for idx := 0; idx < 64; idx++ {
		bit := uint(63 - idx)
		if (xb[0]>>bit)&1 == 1 {
			var prod, shifted uint256.Int
			prod.Mul(&result, exp2Magic[idx])
			shifted.Rsh(&prod, 128)
			result.Set(&shifted)
		}
	}

	// result >>= 63 - (x >> 64); the arithmetic shift extracts the (possibly negative)
	// integer part, so the shift amount lies in [0, 127].
	var intPart int256.Int
	intPart.Rsh(x, 64)
	sh := uint(63 - intPart.Int64())

	var final uint256.Int
	final.Rsh(&result, sh)
	if final.Cmp(max64x64u) > 0 {
		return int256.Int{}, ErrOverflow
	}
	res := asI(&final)
	return res, nil
}

// Exp computes e^x in 64.64 fixed point. Reverts on overflow (x >= 2^70); returns 0 for
// x < -2^70. Mirrors ABDKMath64x64.exp: exp_2(x * log2(e)).
func Exp(x *int256.Int) (int256.Int, error) {
	if x.Cmp(exp2Limit) >= 0 {
		return int256.Int{}, ErrOverflow
	}
	if x.Cmp(negExp2Limit) < 0 {
		return int256.Int{}, nil // underflow -> 0
	}

	var prod, t int256.Int
	prod.Mul(x, expConst) // exact: |x| < 2^70, expConst < 2^129, product < 2^199
	t.Rsh(&prod, 128)     // arithmetic shift (x may be negative)
	return Exp2(&t)
}
