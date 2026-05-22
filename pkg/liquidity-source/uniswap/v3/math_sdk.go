package uniswapv3

// Inlined from github.com/KyberNetwork/uniswapv3-sdk-uint256/{constants,utils}.
// Removes the transitive daoleno/uniswap-sdk-core dependency pulled in by the
// constants package (via its exported PercentZero variable).

import (
	"errors"
	"math/bits"
	"slices"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// ---------- Fee constants (from constants/constants.go) ----------

type FeeAmount uint64

const (
	FeeLowest FeeAmount = 100
	FeeLow    FeeAmount = 500
	FeeMedium FeeAmount = 3000
	FeeHigh   FeeAmount = 10000
	Fee80     FeeAmount = 80
	Fee450    FeeAmount = 450
	Fee2500   FeeAmount = 2500
	FeeMax    FeeAmount = 1000000
)

var TickSpacings = map[FeeAmount]int{
	FeeLowest: 1,
	Fee80:     1,
	FeeLow:    10,
	Fee450:    10,
	FeeMedium: 60,
	Fee2500:   60,
	FeeHigh:   200,
}

// ---------- Tick / sqrt-price constants (from utils/tick_math.go) ----------

const (
	MinTick = -887272
	MaxTick = -MinTick
)

var (
	MinSqrtRatioU256 = uint256.NewInt(4295128739)
	MaxSqrtRatioU256 = uint256.MustFromDecimal("1461446703485210103287273052203988822378723970342")
	q32U256          = uint256.NewInt(1 << 32)
	q96U256          = new(uint256.Int).Exp(uint256.NewInt(2), uint256.NewInt(96))
	maxUint256       = uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	uint128Max       = uint256.MustFromHex("0xffffffffffffffffffffffffffffffff")
	uint160Max       = uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffff")
)

// ---------- Errors ----------

var (
	errExceedMaxInt256       = errors.New("exceed max int256")
	errOverflowUint128       = errors.New("overflow uint128")
	errOverflowUint160       = errors.New("overflow uint160")
	errInvalidTick           = errors.New("invalid tick")
	errInvalidSqrtRatio      = errors.New("invalid sqrt ratio")
	errMulDivOverflow        = errors.New("muldiv overflow")
	errInvalidInput          = errors.New("invalid input")
	errSqrtPriceLessThanZero = errors.New("sqrt price less than zero")
	errLiquidityLessThanZero = errors.New("liquidity less than zero")
	errInvariant             = errors.New("invariant violation")
	errAddOverflow           = errors.New("add overflow")
)

// ---------- 512-bit helpers (from utils/uint256.go) ----------

func umul(x, y *uint256.Int) [8]uint64 {
	var (
		res                           [8]uint64
		carry, carry4, carry5, carry6 uint64
		res1, res2, res3, res4, res5  uint64
	)
	carry, res[0] = bits.Mul64(x[0], y[0])
	carry, res1 = umulHop(carry, x[1], y[0])
	carry, res2 = umulHop(carry, x[2], y[0])
	carry4, res3 = umulHop(carry, x[3], y[0])

	carry, res[1] = umulHop(res1, x[0], y[1])
	carry, res2 = umulStep(res2, x[1], y[1], carry)
	carry, res3 = umulStep(res3, x[2], y[1], carry)
	carry5, res4 = umulStep(carry4, x[3], y[1], carry)

	carry, res[2] = umulHop(res2, x[0], y[2])
	carry, res3 = umulStep(res3, x[1], y[2], carry)
	carry, res4 = umulStep(res4, x[2], y[2], carry)
	carry6, res5 = umulStep(carry5, x[3], y[2], carry)

	carry, res[3] = umulHop(res3, x[0], y[3])
	carry, res[4] = umulStep(res4, x[1], y[3], carry)
	carry, res[5] = umulStep(res5, x[2], y[3], carry)
	res[7], res[6] = umulStep(carry6, x[3], y[3], carry)
	return res
}

func umulHop(z, x, y uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(x, y)
	lo, carry := bits.Add64(lo, z, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	return hi, lo
}

func umulStep(z, x, y, carry uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(x, y)
	lo, carry = bits.Add64(lo, carry, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	lo, carry = bits.Add64(lo, z, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	return hi, lo
}

func udivrem(quot, u []uint64, d *uint256.Int) (rem uint256.Int) {
	var dLen int
	for i := len(d) - 1; i >= 0; i-- {
		if d[i] != 0 {
			dLen = i + 1
			break
		}
	}
	shift := uint(bits.LeadingZeros64(d[dLen-1]))

	var dnStorage uint256.Int
	dn := dnStorage[:dLen]
	for i := dLen - 1; i > 0; i-- {
		dn[i] = (d[i] << shift) | (d[i-1] >> (64 - shift))
	}
	dn[0] = d[0] << shift

	var uLen int
	for i, v := range slices.Backward(u) {
		if v != 0 {
			uLen = i + 1
			break
		}
	}
	if uLen < dLen {
		copy(rem[:], u)
		return rem
	}

	var unStorage [9]uint64
	un := unStorage[:uLen+1]
	un[uLen] = u[uLen-1] >> (64 - shift)
	for i := uLen - 1; i > 0; i-- {
		un[i] = (u[i] << shift) | (u[i-1] >> (64 - shift))
	}
	un[0] = u[0] << shift

	if dLen == 1 {
		r := udivremBy1(quot, un, dn[0])
		rem.SetUint64(r >> shift)
		return rem
	}
	udivremKnuth(quot, un, dn)
	for i := 0; i < dLen-1; i++ {
		rem[i] = (un[i] >> shift) | (un[i+1] << (64 - shift))
	}
	rem[dLen-1] = un[dLen-1] >> shift
	return rem
}

func udivremBy1(quot, u []uint64, d uint64) (rem uint64) {
	reciprocal := reciprocal2by1(d)
	rem = u[len(u)-1]
	for j := len(u) - 2; j >= 0; j-- {
		quot[j], rem = udivrem2by1(rem, u[j], d, reciprocal)
	}
	return rem
}

func reciprocal2by1(d uint64) uint64 {
	reciprocal, _ := bits.Div64(^d, ^uint64(0), d)
	return reciprocal
}

func udivrem2by1(uh, ul, d, reciprocal uint64) (quot, rem uint64) {
	qh, ql := bits.Mul64(reciprocal, uh)
	ql, carry := bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, carry)
	qh++
	r := ul - qh*d
	if r > ql {
		qh--
		r += d
	}
	if r >= d {
		qh++
		r -= d
	}
	return qh, r
}

func udivremKnuth(quot, u, d []uint64) {
	dh := d[len(d)-1]
	dl := d[len(d)-2]
	reciprocal := reciprocal2by1(dh)
	for j := len(u) - len(d) - 1; j >= 0; j-- {
		u2 := u[j+len(d)]
		u1 := u[j+len(d)-1]
		u0 := u[j+len(d)-2]
		var qhat, rhat uint64
		if u2 >= dh {
			qhat = ^uint64(0)
		} else {
			qhat, rhat = udivrem2by1(u2, u1, dh, reciprocal)
			ph, pl := bits.Mul64(qhat, dl)
			if ph > rhat || (ph == rhat && pl > u0) {
				qhat--
			}
		}
		borrow := subMulTo(u[j:], d, qhat)
		u[j+len(d)] = u2 - borrow
		if u2 < borrow {
			qhat--
			u[j+len(d)] += addTo(u[j:], d)
		}
		quot[j] = qhat
	}
}

func subMulTo(x, y []uint64, multiplier uint64) uint64 {
	var borrow uint64
	for i := range y {
		s, carry1 := bits.Sub64(x[i], borrow, 0)
		ph, pl := bits.Mul64(y[i], multiplier)
		t, carry2 := bits.Sub64(s, pl, 0)
		x[i] = t
		borrow = ph + carry1 + carry2
	}
	return borrow
}

func addTo(x, y []uint64) uint64 {
	var carry uint64
	for i := range y {
		x[i], carry = bits.Add64(x[i], y[i], carry)
	}
	return carry
}

// ---------- MulDiv (from utils/full_math.go) ----------

func MulDivRoundingUpV2(a, b, denominator, result *uint256.Int) error {
	var remainder uint256.Int
	if err := MulDivV2(a, b, denominator, result, &remainder); err != nil {
		return err
	}
	if !remainder.IsZero() {
		if result.Cmp(maxUint256) == 0 {
			return errInvariant
		}
		result.AddUint64(result, 1)
	}
	return nil
}

func MulDivV2(x, y, d, z, r *uint256.Int) error {
	if x.IsZero() || y.IsZero() || d.IsZero() {
		z.Clear()
		return nil
	}
	p := umul(x, y)
	var quot [8]uint64
	rem := udivrem(quot[:], p[:], d)
	if r != nil {
		r.Set(&rem)
	}
	copy(z[:], quot[:4])
	if (quot[4] | quot[5] | quot[6] | quot[7]) != 0 {
		return errMulDivOverflow
	}
	return nil
}

func DivRoundingUp(a, denominator, result *uint256.Int) {
	var rem uint256.Int
	result.DivMod(a, denominator, &rem)
	if !rem.IsZero() {
		result.AddUint64(result, 1)
	}
}

// ---------- Most significant bit (from utils/most_significant_bit.go) ----------

type powerOf2 struct {
	power uint
	value *uint256.Int
}

var powersOf2 = []powerOf2{
	{128, uint256.MustFromHex("0x100000000000000000000000000000000")},
	{64, uint256.MustFromHex("0x10000000000000000")},
	{32, uint256.MustFromHex("0x100000000")},
	{16, uint256.MustFromHex("0x10000")},
	{8, uint256.MustFromHex("0x100")},
	{4, uint256.MustFromHex("0x10")},
	{2, uint256.MustFromHex("0x4")},
	{1, uint256.MustFromHex("0x2")},
}

func MostSignificantBit(x *uint256.Int) (uint, error) {
	if x.IsZero() {
		return 0, errInvalidInput
	}
	var tmpX uint256.Int
	tmpX.Set(x)
	var msb uint
	for _, p := range powersOf2 {
		if tmpX.Cmp(p.value) >= 0 {
			tmpX.Rsh(&tmpX, p.power)
			msb += p.power
		}
	}
	return msb, nil
}

// ---------- Tick math (from utils/tick_math.go) ----------

var (
	sqrtConst1  = uint256.MustFromHex("0xfffcb933bd6fad37aa2d162d1a594001")
	sqrtConst2  = uint256.MustFromHex("0x100000000000000000000000000000000")
	sqrtConst3  = uint256.MustFromHex("0xfff97272373d413259a46990580e213a")
	sqrtConst4  = uint256.MustFromHex("0xfff2e50f5f656932ef12357cf3c7fdcc")
	sqrtConst5  = uint256.MustFromHex("0xffe5caca7e10e4e61c3624eaa0941cd0")
	sqrtConst6  = uint256.MustFromHex("0xffcb9843d60f6159c9db58835c926644")
	sqrtConst7  = uint256.MustFromHex("0xff973b41fa98c081472e6896dfb254c0")
	sqrtConst8  = uint256.MustFromHex("0xff2ea16466c96a3843ec78b326b52861")
	sqrtConst9  = uint256.MustFromHex("0xfe5dee046a99a2a811c461f1969c3053")
	sqrtConst10 = uint256.MustFromHex("0xfcbe86c7900a88aedcffc83b479aa3a4")
	sqrtConst11 = uint256.MustFromHex("0xf987a7253ac413176f2b074cf7815e54")
	sqrtConst12 = uint256.MustFromHex("0xf3392b0822b70005940c7a398e4b70f3")
	sqrtConst13 = uint256.MustFromHex("0xe7159475a2c29b7443b29c7fa6e889d9")
	sqrtConst14 = uint256.MustFromHex("0xd097f3bdfd2022b8845ad8f792aa5825")
	sqrtConst15 = uint256.MustFromHex("0xa9f746462d870fdf8a65dc1f90e061e5")
	sqrtConst16 = uint256.MustFromHex("0x70d869a156d2a1b890bb3df62baf32f7")
	sqrtConst17 = uint256.MustFromHex("0x31be135f97d08fd981231505542fcfa6")
	sqrtConst18 = uint256.MustFromHex("0x9aa508b5b7a84e1c677de54f3e99bc9")
	sqrtConst19 = uint256.MustFromHex("0x5d6af8dedb81196699c329225ee604")
	sqrtConst20 = uint256.MustFromHex("0x2216e584f5fa1ea926041bedfe98")
	sqrtConst21 = uint256.MustFromHex("0x48a170391f7dc42444e8fa2")
)

func mulShift(val *uint256.Int, mulBy *uint256.Int) {
	var tmp uint256.Int
	val.Rsh(tmp.Mul(val, mulBy), 128)
}

func GetSqrtRatioAtTick(tick int, result *uint256.Int) error {
	if tick < MinTick || tick > MaxTick {
		return errInvalidTick
	}
	absTick := tick
	if tick < 0 {
		absTick = -tick
	}
	var ratio uint256.Int
	if absTick&0x1 != 0 {
		ratio.Set(sqrtConst1)
	} else {
		ratio.Set(sqrtConst2)
	}
	if (absTick & 0x2) != 0 {
		mulShift(&ratio, sqrtConst3)
	}
	if (absTick & 0x4) != 0 {
		mulShift(&ratio, sqrtConst4)
	}
	if (absTick & 0x8) != 0 {
		mulShift(&ratio, sqrtConst5)
	}
	if (absTick & 0x10) != 0 {
		mulShift(&ratio, sqrtConst6)
	}
	if (absTick & 0x20) != 0 {
		mulShift(&ratio, sqrtConst7)
	}
	if (absTick & 0x40) != 0 {
		mulShift(&ratio, sqrtConst8)
	}
	if (absTick & 0x80) != 0 {
		mulShift(&ratio, sqrtConst9)
	}
	if (absTick & 0x100) != 0 {
		mulShift(&ratio, sqrtConst10)
	}
	if (absTick & 0x200) != 0 {
		mulShift(&ratio, sqrtConst11)
	}
	if (absTick & 0x400) != 0 {
		mulShift(&ratio, sqrtConst12)
	}
	if (absTick & 0x800) != 0 {
		mulShift(&ratio, sqrtConst13)
	}
	if (absTick & 0x1000) != 0 {
		mulShift(&ratio, sqrtConst14)
	}
	if (absTick & 0x2000) != 0 {
		mulShift(&ratio, sqrtConst15)
	}
	if (absTick & 0x4000) != 0 {
		mulShift(&ratio, sqrtConst16)
	}
	if (absTick & 0x8000) != 0 {
		mulShift(&ratio, sqrtConst17)
	}
	if (absTick & 0x10000) != 0 {
		mulShift(&ratio, sqrtConst18)
	}
	if (absTick & 0x20000) != 0 {
		mulShift(&ratio, sqrtConst19)
	}
	if (absTick & 0x40000) != 0 {
		mulShift(&ratio, sqrtConst20)
	}
	if (absTick & 0x80000) != 0 {
		mulShift(&ratio, sqrtConst21)
	}
	if tick > 0 {
		result.Div(maxUint256, &ratio)
		ratio.Set(result)
	}
	var rem uint256.Int
	result.DivMod(&ratio, q32U256, &rem)
	if !rem.IsZero() {
		result.AddUint64(result, 1)
	}
	return nil
}

var (
	magicSqrt10001 = int256.MustFromDec("255738958999603826347141")
	magicTickLow   = int256.MustFromDec("3402992956809132418596140100660247210")
	magicTickHigh  = int256.MustFromDec("291339464771989622907027621153398088495")
)

func GetTickAtSqrtRatio(sqrtRatioX96 *uint256.Int) (int, error) {
	if sqrtRatioX96.Cmp(MinSqrtRatioU256) < 0 || sqrtRatioX96.Cmp(MaxSqrtRatioU256) >= 0 {
		return 0, errInvalidSqrtRatio
	}
	var sqrtRatioX128 uint256.Int
	sqrtRatioX128.Lsh(sqrtRatioX96, 32)
	msb, err := MostSignificantBit(&sqrtRatioX128)
	if err != nil {
		return 0, err
	}
	var r uint256.Int
	if msb >= 128 {
		r.Rsh(&sqrtRatioX128, msb-127)
	} else {
		r.Lsh(&sqrtRatioX128, 127-msb)
	}

	log2 := int256.NewInt(int64(msb - 128))
	log2.Lsh(log2, 64)

	var tmp, f uint256.Int
	for i := range 14 {
		tmp.Mul(&r, &r)
		r.Rsh(&tmp, 127)
		f.Rsh(&r, 128)
		tmp.Lsh(&f, uint(63-i))
		log2.Or(log2, (*int256.Int)(&tmp))
		r.Rsh(&r, uint(f.Uint64()))
	}

	var logSqrt10001, tmp1, tmp2 int256.Int
	logSqrt10001.Mul(log2, magicSqrt10001)
	tickLow := tmp2.Rsh(tmp1.Sub(&logSqrt10001, magicTickLow), 128).Uint64()
	tickHigh := tmp2.Rsh(tmp1.Add(&logSqrt10001, magicTickHigh), 128).Uint64()

	if tickLow == tickHigh {
		return int(tickLow), nil
	}
	var sqrtRatio uint256.Int
	if err = GetSqrtRatioAtTick(int(tickHigh), &sqrtRatio); err != nil {
		return 0, err
	}
	if sqrtRatio.Cmp(sqrtRatioX96) <= 0 {
		return int(tickHigh), nil
	}
	return int(tickLow), nil
}

// ---------- Sqrt price math (from utils/sqrtprice_math.go) ----------

var maxUint160 = uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffff")

func multiplyIn256(x, y, product *uint256.Int) {
	product.Mul(x, y)
}

func addIn256(x, y, sum *uint256.Int) {
	sum.Add(x, y)
}

func GetAmount0DeltaV2(sqrtRatioAX96, sqrtRatioBX96 *uint256.Int, liquidity *uint256.Int, roundUp bool, result *uint256.Int) error {
	if sqrtRatioAX96.Cmp(sqrtRatioBX96) > 0 {
		sqrtRatioAX96, sqrtRatioBX96 = sqrtRatioBX96, sqrtRatioAX96
	}
	var numerator1, numerator2 uint256.Int
	numerator1.Lsh(liquidity, 96)
	numerator2.Sub(sqrtRatioBX96, sqrtRatioAX96)
	if roundUp {
		var deno uint256.Int
		if err := MulDivRoundingUpV2(&numerator1, &numerator2, sqrtRatioBX96, &deno); err != nil {
			return err
		}
		DivRoundingUp(&deno, sqrtRatioAX96, result)
		return nil
	}
	var tmp uint256.Int
	if err := MulDivV2(&numerator1, &numerator2, sqrtRatioBX96, &tmp, nil); err != nil {
		return err
	}
	result.Div(&tmp, sqrtRatioAX96)
	return nil
}

func GetAmount1DeltaV2(sqrtRatioAX96, sqrtRatioBX96 *uint256.Int, liquidity *uint256.Int, roundUp bool, result *uint256.Int) error {
	if sqrtRatioAX96.Cmp(sqrtRatioBX96) > 0 {
		sqrtRatioAX96, sqrtRatioBX96 = sqrtRatioBX96, sqrtRatioAX96
	}
	var diff uint256.Int
	diff.Sub(sqrtRatioBX96, sqrtRatioAX96)
	if roundUp {
		return MulDivRoundingUpV2(liquidity, &diff, q96U256, result)
	}
	return MulDivV2(liquidity, &diff, q96U256, result, nil)
}

func GetNextSqrtPriceFromInput(sqrtPX96 *uint256.Int, liquidity *uint256.Int, amountIn *uint256.Int, zeroForOne bool, result *uint256.Int) error {
	if sqrtPX96.Sign() <= 0 {
		return errSqrtPriceLessThanZero
	}
	if liquidity.Sign() <= 0 {
		return errLiquidityLessThanZero
	}
	if zeroForOne {
		return getNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amountIn, true, result)
	}
	return getNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amountIn, true, result)
}

func GetNextSqrtPriceFromOutput(sqrtPX96 *uint256.Int, liquidity *uint256.Int, amountOut *uint256.Int, zeroForOne bool, result *uint256.Int) error {
	if sqrtPX96.Sign() <= 0 {
		return errSqrtPriceLessThanZero
	}
	if liquidity.Sign() <= 0 {
		return errLiquidityLessThanZero
	}
	if zeroForOne {
		return getNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amountOut, false, result)
	}
	return getNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amountOut, false, result)
}

func getNextSqrtPriceFromAmount0RoundingUp(sqrtPX96 *uint256.Int, liquidity *uint256.Int, amount *uint256.Int, add bool, result *uint256.Int) error {
	if amount.IsZero() {
		result.Set(sqrtPX96)
		return nil
	}
	var numerator1, denominator, product, tmp uint256.Int
	numerator1.Lsh(liquidity, 96)
	multiplyIn256(amount, sqrtPX96, &product)
	if add {
		if tmp.Div(&product, amount).Cmp(sqrtPX96) == 0 {
			addIn256(&numerator1, &product, &denominator)
			if denominator.Cmp(&numerator1) >= 0 {
				return MulDivRoundingUpV2(&numerator1, sqrtPX96, &denominator, result)
			}
		}
		tmp.Div(&numerator1, sqrtPX96)
		tmp.Add(&tmp, amount)
		DivRoundingUp(&numerator1, &tmp, result)
		return nil
	}
	if tmp.Div(&product, amount).Cmp(sqrtPX96) != 0 {
		return errInvariant
	}
	if numerator1.Cmp(&product) <= 0 {
		return errInvariant
	}
	denominator.Sub(&numerator1, &product)
	return MulDivRoundingUpV2(&numerator1, sqrtPX96, &denominator, result)
}

func getNextSqrtPriceFromAmount1RoundingDown(sqrtPX96 *uint256.Int, liquidity *uint256.Int, amount *uint256.Int, add bool, result *uint256.Int) error {
	if add {
		var quotient, tmp uint256.Int
		if amount.Cmp(maxUint160) <= 0 {
			tmp.Lsh(amount, 96)
			quotient.Div(&tmp, liquidity)
		} else {
			tmp.Mul(amount, q96U256)
			quotient.Div(&tmp, liquidity)
		}
		_, overflow := quotient.AddOverflow(&quotient, sqrtPX96)
		if overflow {
			return errAddOverflow
		}
		if quotient.Cmp(uint160Max) > 0 {
			return errOverflowUint160
		}
		result.Set(&quotient)
		return nil
	}
	var quotient uint256.Int
	if err := MulDivRoundingUpV2(amount, q96U256, liquidity, &quotient); err != nil {
		return err
	}
	if sqrtPX96.Cmp(&quotient) <= 0 {
		return errInvariant
	}
	quotient.Sub(sqrtPX96, &quotient)
	result.Set(&quotient)
	return nil
}

// ---------- Swap math (from utils/swap_math.go) ----------

const maxFeeInt = 1000000

var maxFeeUint256 = uint256.NewInt(maxFeeInt)

func ComputeSwapStep(
	sqrtRatioCurrentX96, sqrtRatioTargetX96 *uint256.Int,
	liquidity *uint256.Int,
	amountRemaining *int256.Int,
	feePips FeeAmount,
	sqrtRatioNextX96 *uint256.Int, amountIn, amountOut, feeAmount *uint256.Int,
) error {
	zeroForOne := sqrtRatioCurrentX96.Cmp(sqrtRatioTargetX96) >= 0
	exactIn := amountRemaining.Sign() >= 0

	var amountRemainingU uint256.Int
	toUInt256(amountRemaining, &amountRemainingU)
	if !exactIn {
		amountRemainingU.Neg(&amountRemainingU)
	}

	var maxFeeMinusFeePips uint256.Int
	maxFeeMinusFeePips.SetUint64(maxFeeInt - uint64(feePips))

	if exactIn {
		var amountRemainingLessFee, tmp uint256.Int
		tmp.Mul(&amountRemainingU, &maxFeeMinusFeePips)
		amountRemainingLessFee.Div(&tmp, maxFeeUint256)
		if zeroForOne {
			if err := GetAmount0DeltaV2(sqrtRatioTargetX96, sqrtRatioCurrentX96, liquidity, true, amountIn); err != nil {
				return err
			}
		} else {
			if err := GetAmount1DeltaV2(sqrtRatioCurrentX96, sqrtRatioTargetX96, liquidity, true, amountIn); err != nil {
				return err
			}
		}
		if amountRemainingLessFee.Cmp(amountIn) >= 0 {
			sqrtRatioNextX96.Set(sqrtRatioTargetX96)
		} else {
			if err := GetNextSqrtPriceFromInput(sqrtRatioCurrentX96, liquidity, &amountRemainingLessFee, zeroForOne, sqrtRatioNextX96); err != nil {
				return err
			}
		}
	} else {
		if zeroForOne {
			if err := GetAmount1DeltaV2(sqrtRatioTargetX96, sqrtRatioCurrentX96, liquidity, false, amountOut); err != nil {
				return err
			}
		} else {
			if err := GetAmount0DeltaV2(sqrtRatioCurrentX96, sqrtRatioTargetX96, liquidity, false, amountOut); err != nil {
				return err
			}
		}
		if amountRemainingU.Cmp(amountOut) >= 0 {
			sqrtRatioNextX96.Set(sqrtRatioTargetX96)
		} else {
			if err := GetNextSqrtPriceFromOutput(sqrtRatioCurrentX96, liquidity, &amountRemainingU, zeroForOne, sqrtRatioNextX96); err != nil {
				return err
			}
		}
	}

	max := sqrtRatioTargetX96.Cmp(sqrtRatioNextX96) == 0

	if zeroForOne {
		if !max || !exactIn {
			if err := GetAmount0DeltaV2(sqrtRatioNextX96, sqrtRatioCurrentX96, liquidity, true, amountIn); err != nil {
				return err
			}
		}
		if !max || exactIn {
			if err := GetAmount1DeltaV2(sqrtRatioNextX96, sqrtRatioCurrentX96, liquidity, false, amountOut); err != nil {
				return err
			}
		}
	} else {
		if !max || !exactIn {
			if err := GetAmount1DeltaV2(sqrtRatioCurrentX96, sqrtRatioNextX96, liquidity, true, amountIn); err != nil {
				return err
			}
		}
		if !max || exactIn {
			if err := GetAmount0DeltaV2(sqrtRatioCurrentX96, sqrtRatioNextX96, liquidity, false, amountOut); err != nil {
				return err
			}
		}
	}

	if !exactIn && amountOut.Cmp(&amountRemainingU) > 0 {
		amountOut.Set(&amountRemainingU)
	}
	if exactIn && sqrtRatioNextX96.Cmp(sqrtRatioTargetX96) != 0 {
		feeAmount.Sub(&amountRemainingU, amountIn)
	} else {
		if err := MulDivRoundingUpV2(amountIn, uint256.NewInt(uint64(feePips)), &maxFeeMinusFeePips, feeAmount); err != nil {
			return err
		}
	}
	return nil
}

// ---------- Int type helpers (from utils/int_types.go) ----------

func ToInt256(value *uint256.Int, result *int256.Int) error {
	if value.Sign() < 0 {
		return errExceedMaxInt256
	}
	var ba [32]byte
	value.WriteToArray32(&ba)
	result.SetBytes32(ba[:])
	return nil
}

func toUInt256(value *int256.Int, result *uint256.Int) {
	var ba [32]byte
	value.WriteToArray32(&ba)
	result.SetBytes32(ba[:])
}

func AddDeltaInPlace(x *uint256.Int, y *int256.Int) error {
	var ba [32]byte
	y.WriteToArray32(&ba)
	var yuint uint256.Int
	yuint.SetBytes32(ba[:])
	x.Add(x, &yuint)
	if x.Cmp(uint128Max) > 0 {
		return errOverflowUint128
	}
	return nil
}
