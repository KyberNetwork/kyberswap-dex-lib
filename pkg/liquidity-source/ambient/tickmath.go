package ambient

import (
	"math/big"

	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func mustU256Dec(s string) uint256.Int {
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("mustU256Dec: " + s)
	}
	v, _ := uint256.FromBig(b)
	return *v
}

var (
	MinTick int32 = -665454
	MaxTick int32 = 831818

	uMinSqrtRatio       = *uint256.NewInt(65538)
	uMaxSqrtRatio       = mustU256Dec("21267430153580247136652501917186561138")
	uMaxSqrtRatioMinus1 = func() uint256.Int { v := uMaxSqrtRatio; v.Sub(&v, u256.U1); return v }()

	// uQ64 = 2^64 (for remainder check in GetSqrtRatioAtTick)
	uQ64Tick = func() uint256.Int { var v uint256.Int; v.SetUint64(1); v.Lsh(&v, 64); return v }()

	uMaxUint256 = func() uint256.Int { var v uint256.Int; v.Not(new(uint256.Int)); return v }()

	// CrocTickMath constants (from Ambient Finance Solidity).
	uTickLog2Mul   = mustU256Dec("255738958999603826347141")
	uTickLowOffset = mustU256Dec("3402992956809132418596140100660247210")
	uTickHiOffset  = mustU256Dec("291339464771989622907027621153398088495")

	uTickMagicFactors = [20]uint256.Int{
		*uint256.MustFromHex("0xfffcb933bd6fad37aa2d162d1a594001"),
		*uint256.MustFromHex("0xfff97272373d413259a46990580e213a"),
		*uint256.MustFromHex("0xfff2e50f5f656932ef12357cf3c7fdcc"),
		*uint256.MustFromHex("0xffe5caca7e10e4e61c3624eaa0941cd0"),
		*uint256.MustFromHex("0xffcb9843d60f6159c9db58835c926644"),
		*uint256.MustFromHex("0xff973b41fa98c081472e6896dfb254c0"),
		*uint256.MustFromHex("0xff2ea16466c96a3843ec78b326b52861"),
		*uint256.MustFromHex("0xfe5dee046a99a2a811c461f1969c3053"),
		*uint256.MustFromHex("0xfcbe86c7900a88aedcffc83b479aa3a4"),
		*uint256.MustFromHex("0xf987a7253ac413176f2b074cf7815e54"),
		*uint256.MustFromHex("0xf3392b0822b70005940c7a398e4b70f3"),
		*uint256.MustFromHex("0xe7159475a2c29b7443b29c7fa6e889d9"),
		*uint256.MustFromHex("0xd097f3bdfd2022b8845ad8f792aa5825"),
		*uint256.MustFromHex("0xa9f746462d870fdf8a65dc1f90e061e5"),
		*uint256.MustFromHex("0x70d869a156d2a1b890bb3df62baf32f7"),
		*uint256.MustFromHex("0x31be135f97d08fd981231505542fcfa6"),
		*uint256.MustFromHex("0x9aa508b5b7a84e1c677de54f3e99bc9"),
		*uint256.MustFromHex("0x5d6af8dedb81196699c329225ee604"),
		*uint256.MustFromHex("0x2216e584f5fa1ea926041bedfe98"),
		*uint256.MustFromHex("0x48a170391f7dc42444e8fa2"),
	}
)

// GetSqrtRatioAtTick returns the Q64.64 sqrt ratio for the given tick (clamped to [MinTick, MaxTick]).
func GetSqrtRatioAtTick(tick int32) uint256.Int {
	if tick < MinTick {
		tick = MinTick
	} else if tick > MaxTick {
		tick = MaxTick
	}

	absTick := uint32(tick)
	if tick < 0 {
		absTick = uint32(-int64(tick))
	}

	var ratio uint256.Int
	if absTick&0x1 != 0 {
		ratio.Set(&uTickMagicFactors[0])
	} else {
		ratio.Set(u256.U2Pow128)
	}

	for i := 1; i < len(uTickMagicFactors); i++ {
		if absTick&(uint32(1)<<uint(i)) != 0 {
			ratio.Mul(&ratio, &uTickMagicFactors[i])
			ratio.Rsh(&ratio, 128)
		}
	}

	if tick > 0 {
		ratio.Div(&uMaxUint256, &ratio)
	}

	var rem uint256.Int
	rem.Mod(&ratio, &uQ64Tick)
	ratio.Rsh(&ratio, 64)
	if !rem.IsZero() {
		ratio.Add(&ratio, u256.U1)
	}
	return ratio
}

// GetTickAtSqrtRatio returns the greatest tick t such that GetSqrtRatioAtTick(t) <= sqrtPriceX64.
func GetTickAtSqrtRatio(sqrtPriceX64 uint256.Int) int32 {
	if sqrtPriceX64.Lt(&uMinSqrtRatio) {
		sqrtPriceX64.Set(&uMinSqrtRatio)
	} else if sqrtPriceX64.Cmp(&uMaxSqrtRatio) >= 0 {
		sqrtPriceX64.Sub(&uMaxSqrtRatio, u256.U1)
	}

	var ratio uint256.Int
	ratio.Lsh(&sqrtPriceX64, 64)

	r := ratio // value copy for MSB computation
	// floor(log2(r)) via BitLen — replaces the original 8-threshold loop.
	msb := uint(r.BitLen()) - 1 // r > 0 guaranteed (MinSqrtRatio > 0)

	var log2 uint256.Int
	if msb >= 128 {
		r.Rsh(&ratio, msb-127)
		var delta uint256.Int
		delta.SetUint64(uint64(msb - 128))
		delta.Lsh(&delta, 64)
		log2.Set(&delta)
	} else {
		r.Lsh(&ratio, 127-msb)
		// log2 = -(128-msb)<<64 as 256-bit two's complement
		var delta uint256.Int
		delta.SetUint64(uint64(128 - msb))
		delta.Lsh(&delta, 64)
		log2.Not(&delta)
		log2.Add(&log2, u256.U1)
	}

	for i := uint(63); ; i-- {
		r.Mul(&r, &r)
		r.Rsh(&r, 127)
		var f, shifted uint256.Int
		f.Rsh(&r, 128)
		shifted.Lsh(&f, i)
		log2.Or(&log2, &shifted)
		r.Rsh(&r, uint(f.Uint64()))
		if i == 50 {
			break
		}
	}

	var logSqrt10001, tmp uint256.Int
	logSqrt10001.Mul(&log2, &uTickLog2Mul)

	tmp.Sub(&logSqrt10001, &uTickLowOffset)
	tickLow := arithRsh128(tmp)

	tmp.Add(&logSqrt10001, &uTickHiOffset)
	tickHi := arithRsh128(tmp)

	tickLow32 := int32(int64(tickLow.Uint64()))
	tickHi32 := int32(int64(tickHi.Uint64()))

	if tickLow32 == tickHi32 {
		return tickLow32
	}
	sqrtCheck := GetSqrtRatioAtTick(tickHi32)
	if sqrtCheck.Cmp(&sqrtPriceX64) <= 0 {
		return tickHi32
	}
	return tickLow32
}

// arithRsh128 performs arithmetic (sign-extending) right shift by 128 bits on a
// 256-bit two's complement integer represented as uint256.Int.
func arithRsh128(x uint256.Int) uint256.Int {
	isNeg := x[3]>>63 != 0 // sign bit before shift
	x.Rsh(&x, 128)
	if isNeg {
		x[2] = ^uint64(0)
		x[3] = ^uint64(0)
	}
	return x
}
