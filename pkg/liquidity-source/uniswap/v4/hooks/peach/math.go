package peach

import (
	"math/bits"

	"github.com/holiman/uint256"
)

// squareScaled retains the full uint160 square (320 bits) and scaled product.
// Like MidBlockFeeMath.sol, no sqrt ratio, tick or intermediate price is truncated.
func squareScaled(price *uint256.Int, scale uint64) (out [8]uint64) {
	var square [8]uint64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			hi, lo := bits.Mul64(price[i], price[j])
			k := i + j
			var carry uint64
			square[k], carry = bits.Add64(square[k], lo, 0)
			square[k+1], carry = bits.Add64(square[k+1], hi, carry)
			for n := k + 2; carry != 0; n++ {
				square[n], carry = bits.Add64(square[n], 0, carry)
			}
		}
	}
	var carry uint64
	for i, word := range square {
		hi, lo := bits.Mul64(word, scale)
		var c uint64
		out[i], c = bits.Add64(lo, carry, 0)
		carry = hi + c // hi < scale; scale <= 1e12 + 1e6; the product fits well within 512 bits.
	}
	return
}

func compare512(a, b [8]uint64) int {
	for i := len(a) - 1; i >= 0; i-- {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

// surcharge is the exact capped integer search in Peach MidBlockFeeMath.surcharge.
// Arguments are validated by Track/BeforeSwap before entering this pure hot path.
func surcharge(open, current *uint256.Int, zeroForOne bool, config MidBlockConfig) uint32 {
	if open.IsZero() || current.Eq(open) || config.Fraction == 0 || config.Cap == 0 {
		return 0
	}
	up := current.Gt(open)
	if up != zeroForOne {
		return 0
	}
	high := (uint64(config.Cap)*feePrecision + uint64(config.Fraction) - 1) / uint64(config.Fraction)
	if high < uint64(config.Threshold) {
		high = uint64(config.Threshold)
	}
	if !up && high > feePrecision {
		high = feePrecision
	}
	a := squareScaled(current, feePrecision)
	var low uint64
	for low < high {
		mid := (low + high + 1) / 2
		scale := feePrecision + mid
		if !up {
			scale = feePrecision - mid
		}
		cmp := compare512(a, squareScaled(open, scale))
		if (up && cmp >= 0) || (!up && cmp <= 0) {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low < uint64(config.Threshold) {
		return 0
	}
	return uint32(min(low*uint64(config.Fraction)/feePrecision, uint64(config.Cap)))
}

func resolveFee(autonomous, floor, cap, poke uint32, active bool) uint32 {
	fee := min(autonomous, cap)
	if active && poke != 0 {
		fee = min(max(poke, max(fee/2, floor)), cap)
	}
	return fee
}
