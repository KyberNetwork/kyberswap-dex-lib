package ambient

import (
	"errors"
	"math/big"
)

var (
	bigMaxSQRTRatio, _ = new(big.Int).SetString("21267430153580247136652501917186561138", 10)
	bigMinSQRTRatio    = big.NewInt(65538)
)

var (
	/// @dev The minimum tick that may be passed to #getSqrtRatioAtTick computed from log base 1.0001 of 2**-96
	MIN_TICK Int24 = -665454
	/// @dev The maximum tick that may be passed to #getSqrtRatioAtTick computed from log base 1.0001 of 2**120
	MAX_TICK Int24 = 831818
)

var (
	/// @dev The minimum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MIN_TICK). The reason we don't set this as min(uint128) is so that single precicion moves represent a small fraction.
	MIN_SQRT_RATIO = big.NewInt(65538)
	/// @dev The maximum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MAX_TICK)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("21267430153580247136652501917186561138", 10)
)

var (
	big0xfffcb933bd6fad37aa2d162d1a594001, _ = new(big.Int).SetString("0xfffcb933bd6fad37aa2d162d1a594001", 16)
	big0xfff97272373d413259a46990580e213a, _ = new(big.Int).SetString("0xfff97272373d413259a46990580e213a", 16)
	big0xfff2e50f5f656932ef12357cf3c7fdcc, _ = new(big.Int).SetString("0xfff2e50f5f656932ef12357cf3c7fdcc", 16)
	big0xffe5caca7e10e4e61c3624eaa0941cd0, _ = new(big.Int).SetString("0xffe5caca7e10e4e61c3624eaa0941cd0", 16)
	big0xffcb9843d60f6159c9db58835c926644, _ = new(big.Int).SetString("0xffcb9843d60f6159c9db58835c926644", 16)
	big0xff973b41fa98c081472e6896dfb254c0, _ = new(big.Int).SetString("0xff973b41fa98c081472e6896dfb254c0", 16)
	big0xff2ea16466c96a3843ec78b326b52861, _ = new(big.Int).SetString("0xff2ea16466c96a3843ec78b326b52861", 16)
	big0xfe5dee046a99a2a811c461f1969c3053, _ = new(big.Int).SetString("0xfe5dee046a99a2a811c461f1969c3053", 16)
	big0xfcbe86c7900a88aedcffc83b479aa3a4, _ = new(big.Int).SetString("0xfcbe86c7900a88aedcffc83b479aa3a4", 16)
	big0xf987a7253ac413176f2b074cf7815e54, _ = new(big.Int).SetString("0xf987a7253ac413176f2b074cf7815e54", 16)
	big0xf3392b0822b70005940c7a398e4b70f3, _ = new(big.Int).SetString("0xf3392b0822b70005940c7a398e4b70f3", 16)
	big0xe7159475a2c29b7443b29c7fa6e889d9, _ = new(big.Int).SetString("0xe7159475a2c29b7443b29c7fa6e889d9", 16)
	big0xd097f3bdfd2022b8845ad8f792aa5825, _ = new(big.Int).SetString("0xd097f3bdfd2022b8845ad8f792aa5825", 16)
	big0xa9f746462d870fdf8a65dc1f90e061e5, _ = new(big.Int).SetString("0xa9f746462d870fdf8a65dc1f90e061e5", 16)
	big0x70d869a156d2a1b890bb3df62baf32f7, _ = new(big.Int).SetString("0x70d869a156d2a1b890bb3df62baf32f7", 16)
	big0x31be135f97d08fd981231505542fcfa6, _ = new(big.Int).SetString("0x31be135f97d08fd981231505542fcfa6", 16)
	big0x9aa508b5b7a84e1c677de54f3e99bc9, _  = new(big.Int).SetString("0x9aa508b5b7a84e1c677de54f3e99bc9", 16)
	big0x5d6af8dedb81196699c329225ee604, _   = new(big.Int).SetString("0x5d6af8dedb81196699c329225ee604", 16)
	big0x2216e584f5fa1ea926041bedfe98, _     = new(big.Int).SetString("0x2216e584f5fa1ea926041bedfe98", 16)
	big0x48a170391f7dc42444e8fa2, _          = new(big.Int).SetString("0x48a170391f7dc42444e8fa2", 16)
	big0x8, _                                = new(big.Int).SetString("0x8", 16)
	big0x20, _                               = new(big.Int).SetString("0x20", 16)
	big0x40, _                               = new(big.Int).SetString("0x40", 16)
	big0x80, _                               = new(big.Int).SetString("0x80", 16)
	big0x200, _                              = new(big.Int).SetString("0x200", 16)
	big0x400, _                              = new(big.Int).SetString("0x400", 16)
	big0x800, _                              = new(big.Int).SetString("0x800", 16)
	big0x1000, _                             = new(big.Int).SetString("0x1000", 16)
	big0x2000, _                             = new(big.Int).SetString("0x2000", 16)
	big0x4000, _                             = new(big.Int).SetString("0x4000", 16)
	big0x8000, _                             = new(big.Int).SetString("0x8000", 16)
	big0x20000, _                            = new(big.Int).SetString("0x20000", 16)
	big0x40000, _                            = new(big.Int).SetString("0x40000", 16)
	big0x80000, _                            = new(big.Int).SetString("0x80000", 16)
	bigMaxUint256, _                         = new(big.Int).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	big1Lsh64, _                             = new(big.Int).SetString("18446744073709551616", 10)
	big0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF, _ = new(big.Int).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
	big0xFFFFFFFFFFFFFFFF, _                 = new(big.Int).SetString("0xFFFFFFFFFFFFFFFF", 16)
	big0xFFFFFFFF, _                         = new(big.Int).SetString("0xFFFFFFFF", 16)
	big0xFFFF, _                             = new(big.Int).SetString("0xFFFF", 16)
	big0xFF, _                               = new(big.Int).SetString("0xFF", 16)
	big0xF, _                                = new(big.Int).SetString("0xF", 16)
	big0x3, _                                = new(big.Int).SetString("0x3", 16)
)

// / @notice Calculates sqrt(1.0001^tick) * 2^64
// / @dev Throws if tick < MIN_TICK or tick > MAX_TICK
// / @param tick The input tick for the above formula
// / @return sqrtPriceX64 A Fixed point Q64.64 number representing the sqrt of the ratio of the two assets (token1/token0)
// / at the given tick
func getSqrtRatioAtTick(tick Int24) (*big.Int, error) {
	//     require(tick >= MIN_TICK && tick <= MAX_TICK);
	if tick < MIN_TICK || tick > MAX_TICK {
		return nil, errors.New("tick out of bound")
	}

	//     uint256 absTick = tick < 0 ? uint256(-int256(tick)) : uint256(int256(tick));
	var absTick *big.Int
	if tick < 0 {
		absTick = big.NewInt(int64(tick))
		absTick.Neg(absTick)
	} else {
		absTick = big.NewInt(int64(tick))
	}

	//     uint256 ratio = absTick & 0x1 != 0 ? 0xfffcb933bd6fad37aa2d162d1a594001 : 0x100000000000000000000000000000000;
	var (
		ratio *big.Int
		tmp   = new(big.Int)
	)
	if tmp.And(absTick, big0x1).Cmp(big0) != 0 {
		ratio.Set(big0xfffcb933bd6fad37aa2d162d1a594001)
	} else {
		ratio.Set(big0x100000000000000000000000000000000)
	}

	//     if (absTick & 0x2 != 0) ratio = (ratio * 0xfff97272373d413259a46990580e213a) >> 128;
	if tmp.And(absTick, big0x2).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xfff97272373d413259a46990580e213a)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x4 != 0) ratio = (ratio * 0xfff2e50f5f656932ef12357cf3c7fdcc) >> 128;
	if tmp.And(absTick, big0x4).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xfff2e50f5f656932ef12357cf3c7fdcc)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x8 != 0) ratio = (ratio * 0xffe5caca7e10e4e61c3624eaa0941cd0) >> 128;
	if tmp.And(absTick, big0x8).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xffe5caca7e10e4e61c3624eaa0941cd0)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x10 != 0) ratio = (ratio * 0xffcb9843d60f6159c9db58835c926644) >> 128;
	if tmp.And(absTick, big0x10).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xffcb9843d60f6159c9db58835c926644)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x20 != 0) ratio = (ratio * 0xff973b41fa98c081472e6896dfb254c0) >> 128;
	if tmp.And(absTick, big0x20).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xff973b41fa98c081472e6896dfb254c0)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x40 != 0) ratio = (ratio * 0xff2ea16466c96a3843ec78b326b52861) >> 128;
	if tmp.And(absTick, big0x40).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xff2ea16466c96a3843ec78b326b52861)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x80 != 0) ratio = (ratio * 0xfe5dee046a99a2a811c461f1969c3053) >> 128;
	if tmp.And(absTick, big0x80).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xfe5dee046a99a2a811c461f1969c3053)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x100 != 0) ratio = (ratio * 0xfcbe86c7900a88aedcffc83b479aa3a4) >> 128;
	if tmp.And(absTick, big0x100).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xfcbe86c7900a88aedcffc83b479aa3a4)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x200 != 0) ratio = (ratio * 0xf987a7253ac413176f2b074cf7815e54) >> 128;
	if tmp.And(absTick, big0x200).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xf987a7253ac413176f2b074cf7815e54)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x400 != 0) ratio = (ratio * 0xf3392b0822b70005940c7a398e4b70f3) >> 128;
	if tmp.And(absTick, big0x400).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xf3392b0822b70005940c7a398e4b70f3)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x800 != 0) ratio = (ratio * 0xe7159475a2c29b7443b29c7fa6e889d9) >> 128;
	if tmp.And(absTick, big0x800).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xe7159475a2c29b7443b29c7fa6e889d9)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x1000 != 0) ratio = (ratio * 0xd097f3bdfd2022b8845ad8f792aa5825) >> 128;
	if tmp.And(absTick, big0x1000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xd097f3bdfd2022b8845ad8f792aa5825)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x2000 != 0) ratio = (ratio * 0xa9f746462d870fdf8a65dc1f90e061e5) >> 128;
	if tmp.And(absTick, big0x2000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0xa9f746462d870fdf8a65dc1f90e061e5)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x4000 != 0) ratio = (ratio * 0x70d869a156d2a1b890bb3df62baf32f7) >> 128;
	if tmp.And(absTick, big0x4000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x70d869a156d2a1b890bb3df62baf32f7)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x8000 != 0) ratio = (ratio * 0x31be135f97d08fd981231505542fcfa6) >> 128;
	if tmp.And(absTick, big0x8000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x31be135f97d08fd981231505542fcfa6)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x10000 != 0) ratio = (ratio * 0x9aa508b5b7a84e1c677de54f3e99bc9) >> 128;
	if tmp.And(absTick, big0x10000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x9aa508b5b7a84e1c677de54f3e99bc9)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x20000 != 0) ratio = (ratio * 0x5d6af8dedb81196699c329225ee604) >> 128;
	if tmp.And(absTick, big0x20000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x5d6af8dedb81196699c329225ee604)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x40000 != 0) ratio = (ratio * 0x2216e584f5fa1ea926041bedfe98) >> 128;
	if tmp.And(absTick, big0x40000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x2216e584f5fa1ea926041bedfe98)
		ratio.Rsh(ratio, 128)
	}
	//     if (absTick & 0x80000 != 0) ratio = (ratio * 0x48a170391f7dc42444e8fa2) >> 128;
	if tmp.And(absTick, big0x80000).Cmp(big0) != 0 {
		ratio.Mul(ratio, big0x48a170391f7dc42444e8fa2)
		ratio.Rsh(ratio, 128)
	}

	//     if (tick > 0) ratio = type(uint256).max / ratio;
	if tick > 0 {
		ratio.Div(bigMaxUint256, ratio)
	}

	// this divides by 1<<64 rounding up to go from a Q128.128 to a Q64.64
	// we then downcast because we know the result always fits within 128 bits due to our tick input constraint
	// we round up in the division so getTickAtSqrtRatio of the output price is always consistent
	//     sqrtPriceX64 = uint128((ratio >> 64) + (ratio % (1 << 64) == 0 ? 0 : 1));
	tmp1 := new(big.Int).Set(ratio)
	tmp1.Rsh(tmp1, 64)

	if ratio.Mod(ratio, big1Lsh64).Cmp(big0) == 0 {
		return tmp1, nil
	}

	return tmp1.Add(tmp1, big1), nil
}

// / @notice Calculates the greatest tick value such that getRatioAtTick(tick) <= ratio
// / @dev Throws in case sqrtPriceX64 < MIN_SQRT_RATIO, as MIN_SQRT_RATIO is the lowest value getRatioAtTick may
// / ever return.
// / @param sqrtPriceX64 The sqrt ratio for which to compute the tick as a Q64.64
// / @return tick The greatest tick for which the ratio is less than or equal to the input ratio
func getTickAtSqrtRatio(sqrtPriceX64 *big.Int) (Int24, error) {
	//     // second inequality must be < because the price can never reach the price at the max tick
	//     require(sqrtPriceX64 >= MIN_SQRT_RATIO && sqrtPriceX64 < MAX_SQRT_RATIO);
	if sqrtPriceX64.Cmp(bigMinSQRTRatio) == -1 || sqrtPriceX64.Cmp(bigMaxSQRTRatio) > -1 {
		return 0, errors.New("sqrtPriceX64 out of range")
	}

	//     uint256 ratio = uint256(sqrtPriceX64) << 64;
	ratio := new(big.Int).Set(sqrtPriceX64)
	ratio.Lsh(ratio, 64)

	//     uint256 r = ratio;
	r := new(big.Int).Set(ratio)

	//     uint256 msb = 0;
	msb := big.NewInt(0)

	//     assembly {
	//         let f := shl(7, gt(r, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 7)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(6, gt(r, 0xFFFFFFFFFFFFFFFF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xFFFFFFFFFFFFFFFF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 6)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(5, gt(r, 0xFFFFFFFF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xFFFFFFFF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 5)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(4, gt(r, 0xFFFF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xFFFF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 4)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(3, gt(r, 0xFF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xFF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 3)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(2, gt(r, 0xF))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0xF) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 2)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := shl(1, gt(r, 0x3))
	//         msb := or(msb, f)
	//         r := shr(f, r)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0x3) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}
		f.Lsh(f, 1)

		msb.Or(msb, f)

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         let f := gt(r, 0x1)
	//         msb := or(msb, f)
	//     }
	{
		var f *big.Int
		if r.Cmp(big0x1) == 1 {
			f = big.NewInt(1)
		} else {
			f = big.NewInt(0)
		}

		msb.Or(msb, f)
	}

	//     if (msb >= 128) r = ratio >> (msb - 127);
	//     else r = ratio << (127 - msb);
	big127 := big.NewInt(127)
	if msb.Cmp(big.NewInt(128)) > -1 {
		tmp := new(big.Int).Set(msb)
		tmp.Sub(tmp, big127)
		r.Rsh(ratio, uint(msb.Uint64()))
	} else {
		tmp := new(big.Int).Set(big127)
		tmp.Sub(tmp, msb)
		r.Lsh(ratio, uint(msb.Uint64()))
	}

	//     int256 log_2 = (int256(msb) - 128) << 64;
	log2 := new(big.Int).Set(msb)
	log2.Sub(log2, big.NewInt(128))
	log2.Lsh(log2, 64)

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(63, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 63))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(62, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 62))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(61, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 61))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(60, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 60))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(59, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 59))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(58, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 58))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(57, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 57))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(56, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 56))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(55, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 55))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(54, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 54))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(53, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 53))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(52, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 52))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(51, f))
	//         r := shr(f, r)
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 51))

		r.Rsh(r, uint(f.Uint64()))
	}

	//     assembly {
	//         r := shr(127, mul(r, r))
	//         let f := shr(128, r)
	//         log_2 := or(log_2, shl(50, f))
	//     }
	{
		tmp := new(big.Int).Mul(r, r)
		r.Rsh(tmp, 127)

		f := new(big.Int).Rsh(r, 128)
		log2.Or(log2, new(big.Int).Lsh(f, 50))
	}

	//     int256 log_sqrt10001 = log_2 * 255738958999603826347141; // 128.128 number
	big255738958999603826347141, _ := new(big.Int).SetString("255738958999603826347141", 10)
	logSqrt10001 := new(big.Int).Mul(log2, big255738958999603826347141)

	//     int24 tickLow = int24((log_sqrt10001 - 3402992956809132418596140100660247210) >> 128);
	big3402992956809132418596140100660247210, _ := new(big.Int).SetString("3402992956809132418596140100660247210", 10)
	bigTickLow := new(big.Int).Set(logSqrt10001)
	bigTickLow.Sub(bigTickLow, big3402992956809132418596140100660247210)
	bigTickLow.Rsh(bigTickLow, 128)
	tickLow := Int24(bigTickLow.Int64())

	//     int24 tickHi = int24((log_sqrt10001 + 291339464771989622907027621153398088495) >> 128);
	big291339464771989622907027621153398088495, _ := new(big.Int).SetString("291339464771989622907027621153398088495", 10)
	bigTickHi := new(big.Int).Set(logSqrt10001)
	bigTickHi.Add(bigTickHi, big291339464771989622907027621153398088495)
	bigTickHi.Rsh(bigTickHi, 128)
	tickHi := Int24(bigTickHi.Int64())

	//     tick = tickLow == tickHi ? tickLow : getSqrtRatioAtTick(tickHi) <= sqrtPriceX64 ? tickHi : tickLow;
	var tick Int24
	if tickLow == tickHi {
		tick = tickLow
	} else {
		sqrtRatioAtTick, err := getSqrtRatioAtTick(tickHi)
		if err != nil {
			return 0, err
		}
		if sqrtRatioAtTick.Cmp(sqrtPriceX64) < 1 {
			tick = tickHi
		} else {
			tick = tickLow
		}
	}

	return tick, nil
}
