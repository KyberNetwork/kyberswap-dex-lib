package bignumber

import (
	"encoding/hex"
	"math"
	"math/big"
	"math/bits"

	"github.com/holiman/uint256"
)

const MaxWords = 256 / bits.UintSize

var (
	// TwoPow128 2^128
	TwoPow128 = new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)

	ZeroBI = big.NewInt(0)
	One    = big.NewInt(1)
	Two    = big.NewInt(2)
	Three  = big.NewInt(3)
	Four   = big.NewInt(4)
	Five   = big.NewInt(5)
	Six    = big.NewInt(6)

	MaxU256Hex, _ = hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

var BONE = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
var BoneFloat, _ = new(big.Float).SetString("1000000000000000000")

// TenPowDecimals calculates 10^decimal
func TenPowDecimals(decimal uint8) *big.Float {
	return big.NewFloat(math.Pow10(int(decimal)))
}

func TenPowInt(decimal uint8) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimal)), nil)
}

func NewBig10(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 10)
	return res
}

func NewBig(s string) (res *big.Int) {
	res, _ = new(big.Int).SetString(s, 0)
	return res
}

// similar to `z.ToBig()` but try to re-use space inside `b` instead of allocating
func FillBig(z *uint256.Int, b *big.Int) {
	switch MaxWords { // Compile-time check.
	case 4: // 64-bit architectures.
		if cap(b.Bits()) < 4 {
			// this will resize b, we can be sure that b will only hold at most MaxU256
			b.SetBytes(MaxU256Hex)
		}
		words := b.Bits()[:4]
		words[0] = big.Word(z[0])
		words[1] = big.Word(z[1])
		words[2] = big.Word(z[2])
		words[3] = big.Word(z[3])
		b.SetBits(words)
	case 8: // 32-bit architectures.
		if cap(b.Bits()) < 8 {
			b.SetBytes(MaxU256Hex)
		}
		words := b.Bits()[:8]
		words[0] = big.Word(z[0])
		words[1] = big.Word(z[0] >> 32)
		words[2] = big.Word(z[1])
		words[3] = big.Word(z[1] >> 32)
		words[4] = big.Word(z[2])
		words[5] = big.Word(z[2] >> 32)
		words[6] = big.Word(z[3])
		words[7] = big.Word(z[3] >> 32)
		b.SetBits(words)
	}
}
