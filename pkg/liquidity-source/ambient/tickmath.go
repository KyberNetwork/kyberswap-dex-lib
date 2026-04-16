package ambient

import "math/big"

var (
	MinTick int32 = -665454
	MaxTick int32 = 831818

	MinSqrtRatio = big.NewInt(65538)
	MaxSqrtRatio = mustBig("21267430153580247136652501917186561138")

	q64  = new(big.Int).Lsh(big.NewInt(1), 64)
	q128 = new(big.Int).Lsh(big.NewInt(1), 128)

	tickMagicFactors = []*big.Int{
		mustBig("0xfffcb933bd6fad37aa2d162d1a594001"),
		mustBig("0xfff97272373d413259a46990580e213a"),
		mustBig("0xfff2e50f5f656932ef12357cf3c7fdcc"),
		mustBig("0xffe5caca7e10e4e61c3624eaa0941cd0"),
		mustBig("0xffcb9843d60f6159c9db58835c926644"),
		mustBig("0xff973b41fa98c081472e6896dfb254c0"),
		mustBig("0xff2ea16466c96a3843ec78b326b52861"),
		mustBig("0xfe5dee046a99a2a811c461f1969c3053"),
		mustBig("0xfcbe86c7900a88aedcffc83b479aa3a4"),
		mustBig("0xf987a7253ac413176f2b074cf7815e54"),
		mustBig("0xf3392b0822b70005940c7a398e4b70f3"),
		mustBig("0xe7159475a2c29b7443b29c7fa6e889d9"),
		mustBig("0xd097f3bdfd2022b8845ad8f792aa5825"),
		mustBig("0xa9f746462d870fdf8a65dc1f90e061e5"),
		mustBig("0x70d869a156d2a1b890bb3df62baf32f7"),
		mustBig("0x31be135f97d08fd981231505542fcfa6"),
		mustBig("0x9aa508b5b7a84e1c677de54f3e99bc9"),
		mustBig("0x5d6af8dedb81196699c329225ee604"),
		mustBig("0x2216e584f5fa1ea926041bedfe98"),
		mustBig("0x48a170391f7dc42444e8fa2"),
	}
)

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 0)
	if !ok {
		panic("bad big.Int literal: " + s)
	}
	return v
}

func GetSqrtRatioAtTick(tick int32) *big.Int {
	if tick < MinTick || tick > MaxTick {
		panic("tick out of range")
	}

	absTick := int64(tick)
	if absTick < 0 {
		absTick = -absTick
	}

	ratio := new(big.Int)
	if absTick&0x1 != 0 {
		ratio.Set(tickMagicFactors[0])
	} else {
		ratio.Set(q128)
	}

	for i := 1; i < len(tickMagicFactors); i++ {
		bit := int64(1) << uint(i)
		if absTick&bit != 0 {
			ratio.Mul(ratio, tickMagicFactors[i])
			ratio.Rsh(ratio, 128)
		}
	}

	if tick > 0 {
		maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
		ratio.Div(maxU256, ratio)
	}

	rem := new(big.Int).Mod(ratio, q64)
	result := new(big.Int).Rsh(ratio, 64)
	if rem.Sign() != 0 {
		result.Add(result, big.NewInt(1))
	}
	return result
}

func GetTickAtSqrtRatio(sqrtPriceX64 *big.Int) int32 {
	if sqrtPriceX64.Cmp(MinSqrtRatio) < 0 || sqrtPriceX64.Cmp(MaxSqrtRatio) >= 0 {
		panic("sqrtPrice out of range")
	}

	ratio := new(big.Int).Lsh(sqrtPriceX64, 64)
	r := new(big.Int).Set(ratio)
	msb := uint(0)

	thresholds := []struct {
		cmp  *big.Int
		bits uint
	}{
		{mustBig("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 7},
		{mustBig("0xFFFFFFFFFFFFFFFF"), 6},
		{mustBig("0xFFFFFFFF"), 5},
		{mustBig("0xFFFF"), 4},
		{mustBig("0xFF"), 3},
		{mustBig("0xF"), 2},
		{mustBig("0x3"), 1},
		{mustBig("0x1"), 0},
	}

	for _, t := range thresholds {
		if r.Cmp(t.cmp) > 0 {
			f := uint(1) << t.bits
			msb |= f
			r.Rsh(r, f)
		}
	}
	// last one: f = gt(r, 0x1)
	if r.Cmp(big.NewInt(1)) > 0 {
		msb |= 1
	}

	if msb >= 128 {
		r.Rsh(ratio, msb-127)
	} else {
		r.Lsh(ratio, 127-msb)
	}

	log2 := new(big.Int).Sub(big.NewInt(int64(msb)), big.NewInt(128))
	log2.Lsh(log2, 64)

	for i := uint(63); i >= 50; i-- {
		r.Mul(r, r)
		r.Rsh(r, 127)
		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, i))
		r.Rsh(r, uint(f.Uint64()))
		if i == 50 {
			break
		}
	}

	logSqrt10001 := new(big.Int).Mul(log2, mustBig("255738958999603826347141"))

	tickLow := new(big.Int).Sub(logSqrt10001, mustBig("3402992956809132418596140100660247210"))
	tickLow.Rsh(tickLow, 128)
	if logSqrt10001.Sign() < 0 || tickLow.Sign() < 0 {
		// handle negative: arithmetic right shift
		tickLow = arithRsh128(new(big.Int).Sub(logSqrt10001, mustBig("3402992956809132418596140100660247210")))
	}

	tickHi := new(big.Int).Add(logSqrt10001, mustBig("291339464771989622907027621153398088495"))
	tickHi = arithRsh128(tickHi)

	if tickLow.Cmp(tickHi) == 0 {
		return int32(tickLow.Int64())
	}
	if GetSqrtRatioAtTick(int32(tickHi.Int64())).Cmp(sqrtPriceX64) <= 0 {
		return int32(tickHi.Int64())
	}
	return int32(tickLow.Int64())
}

func arithRsh128(x *big.Int) *big.Int {
	if x.Sign() >= 0 {
		return new(big.Int).Rsh(x, 128)
	}
	// For negative: floor division by 2^128
	result := new(big.Int).Rsh(new(big.Int).Neg(x), 128)
	result.Neg(result)
	// Check if there was a remainder
	shifted := new(big.Int).Lsh(result, 128)
	if shifted.Cmp(x) != 0 {
		result.Sub(result, big.NewInt(1))
	}
	return result
}
