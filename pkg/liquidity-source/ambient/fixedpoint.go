package ambient

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	// uQ48 = 2^48, uQ128 = 2^128.
	uQ48  = new(uint256.Int).Lsh(u256.U1, 48)
	uQ128 = u256.U2Pow128
)

// MulQ64 sets dst = floor(x * y / 2^64) and returns dst.
func MulQ64(dst, x, y *uint256.Int) *uint256.Int {
	dst.Mul(x, y)
	return dst.Rsh(dst, 64)
}

// DivQ64 sets dst = floor(x * 2^64 / y) and returns dst.
func DivQ64(dst, x, y *uint256.Int) *uint256.Int {
	dst.Lsh(x, 64)
	return dst.Div(dst, y)
}

// MulQ48 sets dst = floor(x * y / 2^48) and returns dst (y is uint64).
func MulQ48(dst, x *uint256.Int, y uint64) *uint256.Int {
	var tmp uint256.Int
	tmp.SetUint64(y)
	dst.Mul(x, &tmp)
	return dst.Rsh(dst, 48)
}

// RecipQ64 sets dst = floor(2^128 / x) and returns dst.
func RecipQ64(dst, x *uint256.Int) *uint256.Int {
	return dst.Div(uQ128, x)
}
